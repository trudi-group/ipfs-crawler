package watcher

import (
	"context"
	"errors"
	"time"

	wantlist "github.com/ipfs/go-bitswap/message"
	pb "github.com/ipfs/go-bitswap/message/pb"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio"
	log "github.com/sirupsen/logrus"
)

const (
	protocolBitswap          protocol.ID = "/ipfs/bitswap/1.2.0"
	protocolBitswapOneOne    protocol.ID = "/ipfs/bitswap/1.1.0"
	protocolBitswapOneZero   protocol.ID = "/ipfs/bitswap/1.0.0"
	protocolBitswapNoVersion protocol.ID = "/ipfs/bitswap"
)

var ProtocolStrings = []protocol.ID{
	protocolBitswap,
	protocolBitswapOneOne,
	protocolBitswapOneZero,
	protocolBitswapNoVersion,
}

const MessageSizeMax = 1 << 22

type BSWorker struct {
	cidIn         chan []cid.Cid
	cidOut        chan []*Event
	h             host.Host
	ctx           context.Context
	feedback      chan *Event
	registrations chan *Registration
}

type Registration struct {
	Type    string
	Channel chan *Event
	Peer    peer.ID
}

func NewBSWorker(foreignHost host.Host) (*BSWorker, error) {
	ctx := context.Background() // TODO fix generation?
	// priv, _, _ := crypto.GenerateKeyPair(crypto.RSA, 2048)
	// opts := []libp2p.Option{libp2p.Identity(priv)}
	// h, err := libp2p.New(ctx, opts...)
	feedback := make(chan *Event, 1024)

	worker := &BSWorker{
		ctx:           ctx,
		h:             foreignHost,
		feedback:      feedback,
		registrations: make(chan *Registration, 16),
	}
	for _, prot := range ProtocolStrings {
		worker.h.SetStreamHandler(prot, worker.handle)
	}
	// go worker.handleEvents(ctx)
	return worker, nil
}

func (w *BSWorker) SetOutput(newChan chan *Event) {
	w.feedback = newChan
}

func (w *BSWorker) GetHost() host.Host {
	return w.h
}

func (w *BSWorker) SetHost(newHost host.Host) {
	w.h = newHost
}

func (w *BSWorker) SetCidOut(out chan []*Event) {
	w.cidOut = out
}

func (w *BSWorker) SetCidIn(feeder chan []cid.Cid) {
	w.cidIn = feeder
}

func (w *BSWorker) Call(remote peer.AddrInfo) {
	cids := <-w.cidIn
	ctx, _ := context.WithTimeout(w.ctx, 60*time.Second)
	log.Debug("starting bs messages")
	stream, err := w.h.NewStream(ctx, remote.ID, ProtocolStrings...)
	if err != nil {
		event := &Event{
			Peer:  remote.ID,
			Error: err,
		}
		w.feedback <- event
		return
	}
	defer stream.Close()
	err = w.queryContent(stream, cids)
	if err != nil {
		event := &Event{
			Peer:  remote.ID,
			Error: err,
		}
		w.feedback <- event
		return
	}
	if err != nil {
		log.Info(err)
	}
	// log.Info(response)
}

func (w *BSWorker) AskPeer(remote peer.AddrInfo, cids []cid.Cid) ([]*Event, error) {
	log.WithField("Remote", remote).Debug("Issuing request")
	ctx, cancel := context.WithTimeout(w.ctx, 60*time.Second) // <- TODO: replace via config
	defer cancel()
	var err error = nil
	err = w.h.Connect(ctx, remote)
	if err != nil {
		log.WithFields(log.Fields{
			"Remote": remote,
			"Error":  err,
		}).Info("Error connecting")
		return nil, err // TODO add logging
	}
	stream, err := w.h.NewStream(ctx, remote.ID, ProtocolStrings...)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	err = w.queryContent(stream, cids)
	if err != nil {
		return nil, err
	}
	response, err := w.collectResponses(stream, cids)
	return response, err
}

func (w *BSWorker) collectResponses(stream network.Stream, cids []cid.Cid) ([]*Event, error) {
	cidMap := make(map[cid.Cid]bool)
	remote := stream.Conn().RemotePeer()
	channel := w.registerPeer(remote)
	defer w.unregisterPeer(remote)
	timeout := 30 * time.Second // TODO set from config
	idleTimer := time.NewTimer(timeout)
	defer idleTimer.Stop()
	out := make([]*Event, 0)
	for {
		select {
		case _ = <-idleTimer.C:
			return out, errors.New("Timeout")
		case event := <-channel:
			idleTimer.Reset(timeout)
			out = append(out, event)
			for _, c := range event.Haves {
				cidMap[c] = true
			}
			for _, c := range event.DontHaves {
				cidMap[c] = true
			}
			if len(cidMap) == len(cids) {
				return out, nil
			}
		}
	}
}

func (w *BSWorker) registerPeer(remote peer.ID) chan *Event {
	feedback := make(chan *Event, 1)
	registration := &Registration{
		Peer:    remote,
		Channel: feedback,
		Type:    "register",
	}
	w.registrations <- registration
	return feedback
}

func (w *BSWorker) unregisterPeer(remote peer.ID) {
	registration := &Registration{
		Peer:    remote,
		Channel: nil,
		Type:    "unregister",
	}
	w.registrations <- registration
}

func (w *BSWorker) handleEvents(ctx context.Context) {
	peerMap := make(map[peer.ID]chan *Event)
	// overflow := []*Event{}
	for {
		select {
		case peerEvent := <-w.registrations:
			if peerEvent.Type == "register" {
				peerMap[peerEvent.Peer] = peerEvent.Channel
			} else if peerEvent.Type == "unregister" {
				delete(peerMap, peerEvent.Peer)
			}
		case msg := <-w.feedback:
			log.Debug("received Message")
			peerMap[msg.Peer] <- msg // TODO: use overflow if peer not found
		}
	}
}

func (w *BSWorker) queryContent(stream network.Stream, cids []cid.Cid) error {
	// craft new WantListMessage
	// Send WantList
	// parse responses
	log.WithField("Remote", stream.Conn().RemotePeer()).Debug("Connected to new client")
	wl := wantlist.New(true)
	for _, c := range cids {
		wl.AddEntry(c, 1, pb.Message_Wantlist_Have, true)
	}
	log.WithFields(log.Fields{
		"Remote":  stream.Conn().RemotePeer(),
		"request": wl.Loggable(),
	}).Debug("Issue Request")
	switch stream.Protocol() {
	case protocolBitswapOneOne, protocolBitswap:
		if err := wl.ToNetV1(stream); err != nil {
			log.WithFields(log.Fields{
				"Remote":  stream.Conn().RemotePeer(),
				"request": wl.Loggable(),
				"Error":   err,
			}).Debug("Error during request")
			return err
		}
	case protocolBitswapOneZero, protocolBitswapNoVersion:
		if err := wl.ToNetV0(stream); err != nil {
			log.WithFields(log.Fields{
				"Remote":  stream.Conn().RemotePeer(),
				"request": wl.Loggable(),
				"Error":   err,
			}).Debug("Error during request")
			return err
		}
	default:
		return errors.New("unknown protocol")
	}
	return nil
}

func (w *BSWorker) handle(stream network.Stream) {
	reader := msgio.NewVarintReaderSize(stream, MessageSizeMax)
	msg, err := wantlist.FromMsgReader(reader)
	event := &Event{
		Peer:      stream.Conn().RemotePeer(),
		Error:     nil,
		Haves:     nil,
		DontHaves: nil,
	}
	if err != nil {
		event.Error = err
	} else {
		event.Haves = msg.Haves()
		event.DontHaves = msg.DontHaves()
	}
	log.WithField("Event", event).Debug("Handle new event")
	w.feedback <- event
	log.WithField("Event", event).Debug("Event handled")
}
