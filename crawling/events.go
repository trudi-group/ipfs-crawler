package crawling

import (
	"github.com/libp2p/go-libp2p/core/peer"
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

func (em *EventManager) Subscribe(eventName string, callback Callback) {
	if _, found := em.events[eventName]; !found {
		em.events[eventName] = make([]Callback, 0)
	}
	em.events[eventName] = append(em.events[eventName], callback)
}

func (em *EventManager) Emit(eventName string, remote peer.AddrInfo) {
	if _, found := em.events[eventName]; found {
		for _, calls := range em.events[eventName] {
			go calls.Call(remote)
		}
	}
}
