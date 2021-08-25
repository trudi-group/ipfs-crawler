package crawling

import (
    peer "github.com/libp2p/go-libp2p-core/peer"
)

type EventManager struct {
    events map[string][]Callback
}

type Callback interface {
    Call(peer.AddrInfo)
}

func NewEventManager() *EventManager {
    return &EventManager{
        events: make(map[string][]Callback),
    }
}

func (self *EventManager) Subscribe(eventName string, callback Callback){
    if _, found := self.events[eventName]; !found {
        self.events[eventName] = make([]Callback, 0)
    }
    self.events[eventName] = append(self.events[eventName], callback)
}

func (self *EventManager) Emit (eventName string, remote peer.AddrInfo) {
    if _, found := self.events[eventName]; found {
        for _, calls := range self.events[eventName] {
            go calls.Call(remote)
        }
    }
}
