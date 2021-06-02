package watcher

import (
    cid "github.com/ipfs/go-cid"
    "context"
    peer "github.com/libp2p/go-libp2p-core/peer"
)

type BSManager struct {
    cids []cid.Cid
    Feedback chan(*Event)
    Tasks chan([]cid.Cid)
    Storage []*Event
    Workers []*BSWorker
}

type Event struct {
    Peer peer.ID
    Error error
    Haves []cid.Cid
    DontHaves []cid.Cid
}

func NewBSManager () *BSManager {
    return &BSManager{
        cids: make([]cid.Cid, 0),
        Feedback: make(chan *Event, 64),
        Tasks: make(chan []cid.Cid),
        Storage: make([]*Event, 0),
        Workers: make([]*BSWorker, 0),
    }
}

func (self *BSManager) AddCid(newCids []cid.Cid) {
    self.cids = append(self.cids, newCids...)
}

func (self *BSManager) AddWorker(worker *BSWorker){
    worker.SetOutput(self.Feedback)
    worker.SetCidIn(self.Tasks)
    self.Workers = append(self.Workers, worker)
}

func (self *BSManager) Start(ctx context.Context){
    for {
        select {
        case <-ctx.Done():
            return
        case event := <- self.Feedback:
            self.Storage = append(self.Storage, event)
        case self.Tasks <- self.cids:
            continue
    }
    }
}

func (self *BSManager) GetReport() []*Event {
    return self.Storage
}
