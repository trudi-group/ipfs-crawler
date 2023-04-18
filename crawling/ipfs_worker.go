package crawling

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-msgio"
	"github.com/libp2p/go-msgio/protoio"
	log "github.com/sirupsen/logrus"
)

// TODO: Refer ConnectionTimeout

// PrefixLimitError signals that we have exhausted the bucket space.
type PrefixLimitError struct {
	msg  string
	peer peer.AddrInfo
}

func (e *PrefixLimitError) Error() string {
	return e.msg
}

// DesyncMillisMax sets a limit in milliseconds on the random backoff performed
// before each request to de-sync.
const DesyncMillisMax int = 500

type WorkerConfig struct {
	ConnectTimeout      time.Duration `yaml:"connect_timeout"`
	ConnectionAttempts  int           `yaml:"connection_attempts"`
	InteractionTimeout  time.Duration `yaml:"interaction_timeout"`
	InteractionAttempts int           `yaml:"interaction_attempts"`
	ProtocolStrings     []protocol.ID `yaml:"protocol_strings"`
	UserAgent           string        `yaml:"user_agent"`
}

// IPFSWorker performs the connection and extracting the dht buckets from remote nodes.
type IPFSWorker struct {
	preimageHandler *PreimageHandler
	host            host.Host
	config          WorkerConfig
	eventManager    *EventManager
	closed          chan struct{}
	closingLock     sync.Mutex
}

// NewWorker initiates a new instance of a crawl worker.
// Initalizes the token bucket used for rate limiting and the necessary RSA keys for IPFS
//
// :param cm: Instance of the crawlmanager the new worker will be attached to
// :param id: ID of the new worker
// :param ctx: context that the new worker will be attached to
// :return: fully initialized worker
func NewIPFSWorker(config WorkerConfig, eventManager *EventManager, preimageHandler *PreimageHandler) (*IPFSWorker, error) {
	w := &IPFSWorker{
		preimageHandler: preimageHandler,
		config:          config,
		eventManager:    eventManager,
		closed:          make(chan struct{}),
	}

	// Init the host, i.e., generate priv key and all that stuff
	priv, _, _ := crypto.GenerateKeyPair(crypto.RSA, 2048)

	// The resource manager expects a limiter, se we create one from our limits.
	limiter := rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)

	// Initialize the resource manager
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		return nil, fmt.Errorf("unable to create resource manager: %w", err)
	}

	// Create libp2p host
	opts := []libp2p.Option{libp2p.Identity(priv), libp2p.ResourceManager(rm), libp2p.UserAgent(config.UserAgent)}
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create libp2p host: %w", err)
	}
	w.host = h

	return w, nil
}

func (w *IPFSWorker) GetHost() host.Host {
	return w.host
}

func (w *IPFSWorker) CrawlPeer(askPeer peer.AddrInfo) (*NodeKnows, error) {
	// Strip local addresses
	publicAddrInfo := stripLocalAddrs(askPeer)
	log.WithFields(log.Fields{
		"destAddr": publicAddrInfo,
	}).Debug("IPFSWorker connecting to")
	// Check if there are an addresses left
	if len(publicAddrInfo.Addrs) == 0 {
		// Nope
		return nil, fmt.Errorf("peer %s has only local adresses", askPeer.ID)
	}

	// Sleep to de-sync
	time.Sleep(time.Duration(rand.Intn(DesyncMillisMax)) * time.Millisecond)

	// Roadmap:
	// 1) Connect to peer
	// 2) Start a new stream = subprotocol exchange
	// 3) Send FindNode(s)
	// 4) Parse response, add to Queue
	ctx, cancel := context.WithTimeout(context.Background(), w.config.ConnectTimeout)
	defer cancel()

	var err error
	for i := 0; i < w.config.ConnectionAttempts; i++ {
		err = w.host.Connect(ctx, publicAddrInfo)
		if err != nil {
			// We couldn't connect to the target peer. This is either because it's unreachable or the context timed out.
			// In that case, we give up and consider the peer as unreachable.
			log.WithFields(log.Fields{
				"err":      err,
				"try":      i + 1,
				"destAddr": publicAddrInfo,
			}).Debug("could not connect")
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	// Create a new stream
	ctx, cancel = context.WithTimeout(context.Background(), w.config.InteractionTimeout)
	defer cancel()
	var dhtStream network.Stream
	for i := 0; i < w.config.InteractionAttempts; i++ {
		dhtStream, err = w.host.NewStream(ctx, publicAddrInfo.ID, w.config.ProtocolStrings...)
		if err != nil {
			// ToDo: Better error handling
			log.WithFields(log.Fields{
				"err":      err,
				"try":      i + 1,
				"destAddr": publicAddrInfo,
			}).Debug("could not open stream")
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	defer dhtStream.Close()

	returnedPeers, err := w.fullNeighborCrawl(dhtStream, publicAddrInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"destAddr": publicAddrInfo,
		}).Debug("unable to crawl peer")
		// If there are still some peers that we learned of then we deal with them in the normal way, despite the error.
		// If there are no peers, there's no hope.
		if len(returnedPeers) == 0 {
			return nil, err
		}
	}

	log.Debug("Fire connected callbacks")
	w.eventManager.Emit("connected", w.host, publicAddrInfo)

	// Get information about the peer from the peerstore.
	var infos peerMetadata
	agentVersion, err := w.host.Peerstore().Get(publicAddrInfo.ID, "AgentVersion")
	if err != nil {
		log.WithError(err).WithField("peer", publicAddrInfo.ID).Debug("unable to get agent version")
	} else {
		av := agentVersion.(string)
		infos.AgentVersion = &av
	}
	infos.DHTProtocol = string(dhtStream.Protocol())
	protocols, err := w.host.Peerstore().GetProtocols(publicAddrInfo.ID)
	if err != nil {
		log.WithError(err).WithField("peer", publicAddrInfo.ID).Debug("unable to get supported protocols")
	} else {
		var p []string
		for _, proto := range protocols {
			p = append(p, string(proto))
		}
		infos.SupportedProtocols = p
	}

	return &NodeKnows{id: publicAddrInfo.ID, knows: returnedPeers, info: infos}, nil
}

// fullNeighborCrawl systematically reads the dht buckets from remote node.
//
// Asks the remote node for the closest peers to a given prefix the remote knows.
// Iterates through the prefixes until no new peers are learned.
// Returns an error if connecting fails, message passing fails, or if prefix space is exhausted.
func (w *IPFSWorker) fullNeighborCrawl(remotePeerStream network.Stream, remotePeerInfo peer.AddrInfo) ([]peer.AddrInfo, error) {
	// Send the FindNode packet. Here it goes.
	// Start with a common prefix length of 0 and successively move to closer IDs until we either
	// learn no new peers or our hard cap for the CPL pre-computation is reached.
	var returnedPeers []peer.AddrInfo
	seenIDs := make(map[peer.ID]bool)
	var newlyLearnedPeers int

	recvReader := msgio.NewVarintReaderSize(remotePeerStream, network.MessageSizeMax)
	defer recvReader.Close()

	var i int
	// Ask at least 4 times
	for i = 0; (i < 4 || newlyLearnedPeers != 0) && (i < 24); i++ {
		newlyLearnedPeers = 0
		target := w.preimageHandler.FindPreImageForCPL(remotePeerInfo, uint8(i))
		log.WithFields(log.Fields{
			"cpl":      i,
			"destAddr": remotePeerInfo,
		}).Trace("Sending FindNode.")

		var (
			peerResponse []peer.AddrInfo
			err          error
		)
		for i := 0; i < w.config.InteractionAttempts; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), w.config.InteractionTimeout)
			defer cancel()
			peerResponse, err = sendFindNode(ctx, recvReader, target, remotePeerStream)
			if err != nil {
				log.WithFields(log.Fields{
					"err":      err,
					"try":      i + 1,
					"destAddr": remotePeerInfo,
				}).Debug("failed to send FIND_NODE")
			} else {
				break
			}
		}
		if err != nil {
			return returnedPeers, err
		}

		for _, p := range peerResponse {
			if _, ok := seenIDs[p.ID]; ok {
				continue
			}
			returnedPeers = append(returnedPeers, p)
			seenIDs[p.ID] = true
			newlyLearnedPeers++
		}
		log.WithFields(log.Fields{
			"numLearnedPeers": newlyLearnedPeers,
		}).Trace("IPFSWorker learned peers.")
	}

	if i == 23 {
		// Return that we reached the prefix limit, so this can be tracked.
		return returnedPeers, &PrefixLimitError{
			msg:  "Prefix limit reached.",
			peer: remotePeerInfo,
		}
	}

	// Everything went well
	return returnedPeers, nil
}

// Stop stops the IPFSWorker.
func (w *IPFSWorker) Stop() error {
	w.closingLock.Lock()
	select {
	case <-w.closed:
		// Already closed
		w.closingLock.Unlock()
		return nil
	default:
		// Not closed
		close(w.closed)
	}
	w.closingLock.Unlock()

	// Close libp2p host
	err := w.host.Close()
	if err != nil {
		return fmt.Errorf("unable to close libp2p host: %w", err)
	}

	return nil
}

// sendFindNode probes the remote node for neighborhood nodes.
// :param ctx: controlling context
// :param recvReader: Reader/parser for the responses
// :param target: the prefix we are interested in
// :param remotePeerStream: Connection to remote node
// :return: list of received peer adresses
func sendFindNode(ctx context.Context, recvReader msgio.Reader, target []byte, remotePeerStream network.Stream) ([]peer.AddrInfo, error) {
	// Send the packet to the target host and wait for the response or context timeout
	err := protoio.NewDelimitedWriter(remotePeerStream).WriteMsg(pb.NewMessage(pb.Message_FIND_NODE, target, 0))
	if err != nil {
		return nil, err
	}

	// Receive the response and handle it accordingly
	var response pb.Message

	// Async-ify ReadMsg
	errChan := make(chan error)
	responseChan := make(chan []byte)
	go func() {
		msgbytes, err := recvReader.ReadMsg()
		if err != nil {
			errChan <- err
		} else {
			responseChan <- msgbytes
		}
	}()

	select {
	case <-ctx.Done():
		// The context timed out, abort sending/receiving and return.
		return nil, ctx.Err()

	case msg := <-responseChan:
		// Parse the request and then signal that the msgbytes-buffer can be used again
		err = response.Unmarshal(msg)
		if err != nil {
			log.WithError(err).Warn("unable to unmarshal FIND_NODE response")
			return nil, err
		}
		recvReader.ReleaseMsg(msg)
		peerInfo := pb.PBPeersToPeerInfos(response.GetCloserPeers())
		var pi []peer.AddrInfo
		for _, p := range peerInfo {
			pi = append(pi, *p)
		}
		return pi, nil

	case err := <-errChan:
		return nil, err
	}
}
