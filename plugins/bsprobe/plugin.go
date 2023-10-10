// Package bsprobe implements a plugin to probe a peer for content via Bitswap.
package bsprobe

import (
	"context"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	bsmsg "github.com/ipfs/go-bitswap/message"
	pb "github.com/ipfs/go-bitswap/message/pb"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	crawlLib "ipfs-crawler/crawling"
)

const (
	protocolBitswapNoVersion protocol.ID = "/ipfs/bitswap"
	protocolBitswapOneOne                = protocolBitswapNoVersion + "/1.1.0"
	protocolBitswapOneZero               = protocolBitswapNoVersion + "/1.0.0"
	protocolBitswapOneTwo                = protocolBitswapNoVersion + "/1.2.0"
)

const pluginName = "bitswap-probe"

// protocolStrings is a collection of all (currently) implemented versions of
// Bitswap we can talk to.
// They are ordered in descending version order, because the newer versions
// support WANT_HAVE, which is much better for our bandwidth.
var protocolStrings = []protocol.ID{
	protocolBitswapOneTwo,
	protocolBitswapOneOne,
	protocolBitswapOneZero,
	protocolBitswapNoVersion,
}

// Config contains the configuration for the plugin.
type Config struct {
	// A list of CIDs to ask for.
	Cids []cid.Cid `yaml:"cids"`

	// Timeout to apply to requests.
	RequestTimeout time.Duration `yaml:"request_timeout"`

	// How long to wait for responses.
	ResponsePeriod time.Duration `yaml:"response_period"`
}

func init() {
	crawlLib.RegisterPlugin(pluginName, driver{})
}

type driver struct{}

func (driver) NewImpl(h host.Host, cfgBytes []byte) (crawlLib.Plugin, error) {
	var cfg Config
	err := yaml.Unmarshal(cfgBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return newProbe(h, cfg)
}

type bitswapMessageResult struct {
	msg bsmsg.BitSwapMessage
	err error
}

type bitswapProbe struct {
	cfg              Config
	wantHaveMessage  bsmsg.Exportable
	wantBlockMessage bsmsg.Exportable
	h                host.Host

	receivers  map[peer.ID]chan bitswapMessageResult
	receiversM sync.Mutex

	shutdownM sync.Mutex
	shutdown  chan struct{}
}

// ProbeResult contains the result of probing a peer for content via Bitswap.
type ProbeResult struct {
	// Whether an error was encountered during receipt of messages.
	// The other fields are still relevant even if this is not nil, since
	// some replies could have been received already.
	Error error `json:"error"`

	// Haves are CIDs for which the peer explicitly stated block presence.
	Haves []cid.Cid `json:"haves"`

	// DontHaves are CIDs for which the peer explicitly stated block absence.
	DontHaves []cid.Cid `json:"dont_haves"`

	// Blocks are CIDs for which the peer sent a block.
	Blocks []cid.Cid `json:"blocks"`

	// NoResponse are CIDs for which the peer did not send a response.
	// If no error was encountered during the receipt of responses, these can be
	// understood as implicit block absences.
	NoResponse []cid.Cid `json:"no_response"`
}

func newProbe(h host.Host, cfg Config) (*bitswapProbe, error) {
	wlHave := bsmsg.New(true)
	wlBlock := bsmsg.New(true)

	for _, c := range cfg.Cids {
		wlHave.AddEntry(c, math.MaxInt32, pb.Message_Wantlist_Have, true)
		wlBlock.AddEntry(c, math.MaxInt32, pb.Message_Wantlist_Block, false)
	}

	worker := &bitswapProbe{
		cfg:              cfg,
		wantHaveMessage:  wlHave,
		wantBlockMessage: wlBlock,
		h:                h,
		receivers:        make(map[peer.ID]chan bitswapMessageResult),
		shutdown:         make(chan struct{}),
	}

	// Register ourselves as handler for Bitswap streams.
	for _, p := range protocolStrings {
		h.SetStreamHandler(p, worker.handleStream)
	}

	return worker, nil
}

func (*bitswapProbe) Name() string {
	return pluginName
}

func (w *bitswapProbe) HandlePeer(remote peer.AddrInfo) (interface{}, error) {
	log.WithField("remote", remote).Debug("querying via Bitswap")

	// TODO does this context apply to sending messages, too? Probably not...
	ctx, cancel := context.WithTimeout(context.Background(), w.cfg.RequestTimeout)
	defer cancel()

	// Open a new Bitswap stream to send the request on.
	stream, err := w.h.NewStream(ctx, remote.ID, protocolStrings...)
	if err != nil {
		return nil, fmt.Errorf("unable to open stream: %w", err)
	}
	defer stream.Close()

	// Let our stream handler know where to direct responses.
	channel := w.registerPeer(remote.ID)
	defer w.unregisterPeer(remote.ID)

	err = w.queryContent(stream)
	if err != nil {
		return nil, fmt.Errorf("unable to query for content: %w", err)
	}

	// TODO do we need to handle responses on the same stream?

	responses := w.collectResponses(remote.ID, channel)
	if responses.Error != nil {
		log.WithError(responses.Error).WithField("remote", remote).Warn("unable to receive responses")
	}
	return responses, nil
}

func (w *bitswapProbe) Shutdown() error {
	w.shutdownM.Lock()
	defer w.shutdownM.Unlock()

	select {
	case <-w.shutdown:
		// already shut down
	default:
		// not yet shut down
		close(w.shutdown)
	}

	// Unregister ourselves as handler for Bitswap streams.
	for _, p := range protocolStrings {
		w.h.RemoveStreamHandler(p)
	}

	return nil
}

func (w *bitswapProbe) collectResponses(remote peer.ID, responses <-chan bitswapMessageResult) ProbeResult {
	outstanding := make(map[cid.Cid]struct{})
	for _, c := range w.cfg.Cids {
		outstanding[c] = struct{}{}
	}

	haves := make(map[cid.Cid]struct{})
	dontHaves := make(map[cid.Cid]struct{})
	blocks := make(map[cid.Cid]struct{})
	var err error

	timeout := time.After(w.cfg.ResponsePeriod)

outer:
	for {
		select {
		case _ = <-timeout:
			break outer
		case res, ok := <-responses:
			if !ok {
				// Channel closed because peer connection was closed.
				break outer
			}
			if res.err != nil {
				err = res.err
				break outer
			}
			for _, b := range res.msg.Blocks() {
				if _, ok := outstanding[b.Cid()]; !ok {
					log.WithField("remote", remote).WithField("cid", b.Cid()).Warn("received duplicate response for CID")
				}
				delete(outstanding, b.Cid())
				blocks[b.Cid()] = struct{}{}
			}
			for _, c := range res.msg.Haves() {
				if _, ok := outstanding[c]; !ok {
					log.WithField("remote", remote).WithField("cid", c).Warn("received duplicate response for CID")
				}
				delete(outstanding, c)
				haves[c] = struct{}{}
			}
			for _, c := range res.msg.DontHaves() {
				if _, ok := outstanding[c]; !ok {
					log.WithField("remote", remote).WithField("cid", c).Warn("received duplicate response for CID")
				}
				delete(outstanding, c)
				dontHaves[c] = struct{}{}
			}

			// Shortcut: if (by some miracle) we have a response for every CID,
			// return early.
			if len(outstanding) == 0 {
				break outer
			}
		}
	}

	var res ProbeResult
	for c := range haves {
		res.Haves = append(res.Haves, c)
	}
	for c := range dontHaves {
		res.DontHaves = append(res.DontHaves, c)
	}
	for c := range blocks {
		res.Blocks = append(res.Blocks, c)
	}
	for c := range outstanding {
		res.NoResponse = append(res.NoResponse, c)
	}
	res.Error = err
	return res
}

func (w *bitswapProbe) registerPeer(remote peer.ID) <-chan bitswapMessageResult {
	feedback := make(chan bitswapMessageResult)

	w.receiversM.Lock()
	defer w.receiversM.Unlock()
	if _, ok := w.receivers[remote]; ok {
		panic("parallel requests to one peer")
	}
	w.receivers[remote] = feedback

	return feedback
}

func (w *bitswapProbe) unregisterPeer(remote peer.ID) {
	w.receiversM.Lock()
	defer w.receiversM.Unlock()
	if _, ok := w.receivers[remote]; !ok {
		// No-op
		return
	}
	delete(w.receivers, remote)
}

func (w *bitswapProbe) receiveError(remote peer.ID, err error) {
	w.receiversM.Lock()
	defer w.receiversM.Unlock()

	// We could still receive after we're not interested in responses
	// anymore, i.e., after we've unregistered the peer. In that case there's
	// nothing to be done.
	if r, ok := w.receivers[remote]; ok {
		// We ignore EOF because that's pretty... normal.
		if err != io.EOF {
			r <- bitswapMessageResult{err: err}
		}
		close(r)
		delete(w.receivers, remote)
	}
}

func (w *bitswapProbe) receiveMsg(remote peer.ID, msg bsmsg.BitSwapMessage) {
	w.receiversM.Lock()
	defer w.receiversM.Unlock()

	// We could still receive after we're not interested in responses
	// anymore, i.e., after we've unregistered the peer. In that case there's
	// nothing to be done.
	if r, ok := w.receivers[remote]; ok {
		r <- bitswapMessageResult{msg: msg}
	}
}

func (w *bitswapProbe) queryContent(stream network.Stream) error {
	remote := stream.Conn().RemotePeer()

	// Set write timeout
	err := stream.SetWriteDeadline(time.Now().Add(w.cfg.RequestTimeout))
	if err != nil {
		log.WithError(err).WithField("remote", remote).Warn("unable to set write deadline on stream")
	}

	// We need to check the protocol this is running on:
	switch stream.Protocol() {
	case protocolBitswapOneTwo:
		// 1.2.0 supports WANT_HAVE and needs the new serialization format
		err = w.wantHaveMessage.ToNetV1(stream)
	case protocolBitswapOneOne:
		// 1.1.0 supports WANT_BLOCK and needs the new format
		err = w.wantBlockMessage.ToNetV1(stream)
	case protocolBitswapOneZero, protocolBitswapNoVersion:
		// 1.0.0 and no-version support WANT_BLOCK and need the old format
		err = w.wantBlockMessage.ToNetV0(stream)
	default:
		panic(fmt.Sprintf("invalid protocol: %s", stream.Protocol()))
	}

	if err != nil {
		log.WithError(err).WithField("remote", remote).Warn("unable to send message")
		return fmt.Errorf("unable to send message: %w", err)
	}

	return nil
}

func (w *bitswapProbe) handleStream(s network.Stream) {
	defer s.Close()

	select {
	case <-w.shutdown:
		_ = s.Reset()
		return
	default:
	}

	remote := s.Conn().RemotePeer()
	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	for {
		received, err := bsmsg.FromMsgReader(reader)
		if err != nil {
			if err != io.EOF {
				_ = s.Reset()

				log.WithField("remote", remote).WithError(err).Debug("handleStream")
			}
			// Propagate Error
			w.receiveError(remote, err)
			return
		}

		w.receiveMsg(remote, received)
	}
}
