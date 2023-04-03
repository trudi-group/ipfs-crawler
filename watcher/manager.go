package watcher

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"
)

type BSManager struct {
	cids     []cid.Cid
	Feedback chan *Event
	Tasks    chan []cid.Cid
	Storage  []*Event
	Workers  []*BSWorker
	logger   *log.Entry
	finished chan bool
}

type Event struct {
	Peer      peer.ID
	Error     error
	Haves     []cid.Cid
	DontHaves []cid.Cid
}

func NewBSManager() *BSManager {
	return &BSManager{
		cids:     make([]cid.Cid, 0),
		Feedback: make(chan *Event, 64),
		Tasks:    make(chan []cid.Cid),
		Storage:  make([]*Event, 0),
		Workers:  make([]*BSWorker, 0),
		logger: log.WithFields(log.Fields{
			"module": "BitswapManager",
		}),
		finished: make(chan bool, 0),
	}
}

func (m *BSManager) AddCid(newCids []cid.Cid) {
	m.cids = append(m.cids, newCids...)
}

func (m *BSManager) AddWorker(worker *BSWorker) {
	worker.SetOutput(m.Feedback)
	worker.SetCidIn(m.Tasks)
	m.Workers = append(m.Workers, worker)
}

func (m *BSManager) Start(ctx context.Context) {
	drainTimeout := 30 * time.Second
	drainTimer := time.NewTimer(drainTimeout)
	for {
		select {
		case <-ctx.Done(): // received signal to shutdown,
			for {
				select {
				case event := <-m.Feedback:
					m.logger.Trace("received new bitswap message")
					drainTimer.Reset(drainTimeout)
					m.Storage = append(m.Storage, event)
				case <-drainTimer.C:
					m.logger.Info("Hit drainTimer")
					m.finished <- true
					return
				}
			}
			return
		case event := <-m.Feedback:
			m.logger.Trace("received new bitswap message")
			drainTimer.Reset(drainTimeout)
			m.Storage = append(m.Storage, event)
		case m.Tasks <- m.cids:
			continue
		}
	}
}

func (m *BSManager) Wait() {
	<-m.finished
}

func (m *BSManager) GetReport() []*Event {
	return deduplicate(m.Storage)
}

func deduplicate(storage []*Event) []*Event {
	dupMap := make(map[peer.ID]*Event)
	out := make([]*Event, 0)
	for _, event := range storage {
		if _, ok := dupMap[event.Peer]; ok {
			dupMap[event.Peer].Haves = append(dupMap[event.Peer].Haves, event.Haves...)
			dupMap[event.Peer].DontHaves = append(dupMap[event.Peer].DontHaves, event.DontHaves...)
		} else {
			dupMap[event.Peer] = event
		}
	}
	for _, val := range dupMap {
		val.Haves = uniqueCids(val.Haves)
		val.DontHaves = uniqueCids(val.DontHaves)
		out = append(out, val)
	}
	return out
}

func uniqueCids(orig []cid.Cid) []cid.Cid {
	var out []cid.Cid
	dupMap := make(map[cid.Cid]bool)
	for _, val := range orig {
		if _, found := dupMap[val]; !found {
			dupMap[val] = true
			out = append(out, val)
		}
	}
	return out
}
