package crawling

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio"
	"github.com/libp2p/go-msgio/protoio"
	log "github.com/sirupsen/logrus"
)

// CrawlerConfig contains the configuration for the crawler.
type CrawlerConfig struct {
	ProtocolStrings []protocol.ID `yaml:"protocol_strings"`

	InteractionTimeout  time.Duration `yaml:"interaction_timeout"`
	InteractionAttempts uint          `yaml:"interaction_attempts"`
}

func (c CrawlerConfig) check() error {
	if len(c.ProtocolStrings) == 0 {
		return fmt.Errorf("missing protocol strings")
	}
	if c.InteractionAttempts <= 0 {
		return fmt.Errorf("missing or invalid interaction attempts")
	}
	if c.InteractionTimeout <= time.Duration(0) {
		return fmt.Errorf("missing interaction timeout")
	}

	return nil
}

type crawler struct {
	config CrawlerConfig

	h               host.Host
	preimageHandler *PreimageHandler

	shutdownM sync.Mutex
	shutdown  chan struct{}
}

func newCrawler(h host.Host, c CrawlerConfig, ph *PreimageHandler) (*crawler, error) {
	err := c.check()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &crawler{
		config:          c,
		h:               h,
		preimageHandler: ph,
		shutdown:        make(chan struct{}),
	}, nil
}

// HandlePeer (almost) implements Plugin, except for the return type.
func (c *crawler) HandlePeer(p peer.AddrInfo) (*crawlData, error) {
	// Roadmap:
	// 1) Start a new stream = subprotocol exchange
	// 2) Send FindNode(s)
	// 3) Parse responses

	// Create a new stream
	var dhtStream network.Stream
	var err error
	for i := uint(0); i < c.config.InteractionAttempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), c.config.InteractionTimeout)
		defer cancel()
		dhtStream, err = c.h.NewStream(ctx, p.ID, c.config.ProtocolStrings...)
		if err != nil {
			log.WithFields(log.Fields{
				"err":    err,
				"try":    i + 1,
				"peerID": p.ID,
			}).Debug("could not open stream")
		} else {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("unable to open stream: %w", err)
	}
	defer func() { _ = dhtStream.Close() }()

	crawlStartedTs := time.Now()
	neighbors, err := c.fullNeighborCrawl(dhtStream, p.ID)
	if err != nil {
		if len(neighbors) == 0 {
			// We got nothing and a lot of things went wrong, might as well report that...
			return nil, fmt.Errorf("failed to extract peers: %w", err)
		}
	}

	// We hide any errors if we got at least some peers.
	// TODO maybe this is not optimal
	return &crawlData{
		neighbors:              neighbors,
		crawlStartedTimestamp:  crawlStartedTs,
		crawlFinishedTimestamp: time.Now(),
	}, nil
}

// fullNeighborCrawl systematically reads the dht buckets from remote node.
//
// Asks the remote node for the closest peers to a given prefix the remote knows.
// Iterates through the prefixes until no new peers are learned.
// Returns an error if connecting fails, or message passing fails entirely.
func (c *crawler) fullNeighborCrawl(s network.Stream, p peer.ID) ([]peer.AddrInfo, error) {
	// Start with a common prefix length of 0 and successively move to closer IDs until we either
	// learn no new peers or our hard cap for the CPL pre-computation is reached.
	var neighbors []peer.AddrInfo
	var err error
	seenIDs := make(map[peer.ID]struct{})

	recvReader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	defer recvReader.Close()

	// We ask at least four times, or until we learn no new peers.
	// TODO we could create parallel streams, one per CPL, and ask concurrently.
	anyNewPeers := false
	for i := 0; i < 4 || (i < MaxCPL && anyNewPeers); i++ {
		anyNewPeers = false
		target := c.preimageHandler.findPreImageForCPL(p, uint8(i))
		log.WithFields(log.Fields{
			"cpl":      i,
			"destAddr": p,
		}).Trace("Sending FindNode.")

		var peerResponse []peer.AddrInfo
		for i := uint(0); i < c.config.InteractionAttempts; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), c.config.InteractionTimeout)
			defer cancel()
			peerResponse, err = sendFindNode(ctx, recvReader, target, s)
			if err != nil {
				log.WithFields(log.Fields{
					"err":      err,
					"try":      i + 1,
					"destAddr": p,
				}).Debug("failed to send FIND_NODE")
			} else {
				break
			}
		}
		if err != nil {
			log.WithError(err).WithField("peer", p).WithField("bucket", i).Debug("failed to crawl bucket")
		} else {
			log.WithField("bucket", i).WithField("peers", peerResponse).WithField("peer", p).Debug("crawled bucket")
		}

		for _, p := range peerResponse {
			if _, ok := seenIDs[p.ID]; ok {
				continue
			}
			seenIDs[p.ID] = struct{}{}
			neighbors = append(neighbors, p)
			anyNewPeers = true
		}
		if anyNewPeers && i == 23 {
			// This is not always an error: if we're too slow and the peer
			// concurrently modifies its routing table, this will be triggered,
			// too.
			log.WithField("peer", p).Debug("prefix limit reached during crawling. Closer buckets are not dumped. Please report this via Github")
		}
	}

	// Everything went well (enough)
	return neighbors, err
}

// sendFindNode probes the remote node for neighborhood nodes.
// :param ctx: controlling context
// :param recvReader: Reader/parser for the responses
// :param target: the prefix we are interested in
// :param remotePeerStream: Connection to remote node
// :return: list of received peer adresses
func sendFindNode(ctx context.Context, recvReader msgio.Reader, target []byte, s network.Stream) ([]peer.AddrInfo, error) {
	// Send the packet to the target host and wait for the response or context timeout
	err := protoio.NewDelimitedWriter(s).WriteMsg(pb.NewMessage(pb.Message_FIND_NODE, target, 0))
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
			select {
			case errChan <- err:
			// All good
			case <-ctx.Done():
				// We need this to clean up the channel
				close(errChan)
			}
		} else {
			select {
			case responseChan <- msgbytes:
			// All good
			case <-ctx.Done():
				// We need to release the message and clean up the channel
				recvReader.ReleaseMsg(msgbytes)
				close(responseChan)
			}
		}
	}()

	select {
	case <-ctx.Done():
		// The context timed out, abort sending/receiving and return.
		return nil, ctx.Err()

	case msg := <-responseChan:
		// We (deliberately) introduce a race condition with the async reader, since we both listen on the context
		// channel. We need to check for that here.
		if ctx.Err() != nil {
			return nil, err
		}

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
		// We (deliberately) introduce a race condition with the async reader, since we both listen on the context
		// channel. We need to check for that here.
		if ctx.Err() != nil {
			return nil, err
		}

		return nil, err
	}
}

func (c *crawler) Shutdown() error {
	c.shutdownM.Lock()
	defer c.shutdownM.Unlock()

	select {
	case <-c.shutdown:
		// already shut down
	default:
		// not yet shut down
		close(c.shutdown)
	}

	// Boilerplate, empty.

	return nil
}
