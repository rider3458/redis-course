package storage

import (
	"container/list"
	"sync"
)

// lru implements EvictionPolicy with exact LRU ordering.
//
// A hash map points each key at its node in a doubly linked list. The front of
// the list holds the most recently used key and the back holds the least
// recently used key, which is the eviction victim. Accessing a key moves its
// node to the front in constant time.
type lru struct {
	mu      sync.Mutex
	order   *list.List
	entries map[string]*list.Element
}

// NewLRU returns a least-recently-used eviction policy.
func NewLRU() EvictionPolicy {
	return &lru{
		order:   list.New(),
		entries: make(map[string]*list.Element),
	}
}

func (p *lru) Name() string { return "lru" }

func (p *lru) OnInsert(key string) {
	p.mu.Lock()
	p.touchLocked(key)
	p.mu.Unlock()
}

func (p *lru) OnAccess(key string) {
	p.mu.Lock()
	p.touchLocked(key)
	p.mu.Unlock()
}

func (p *lru) OnRemove(key string) {
	p.mu.Lock()
	if element, ok := p.entries[key]; ok {
		p.order.Remove(element)
		delete(p.entries, key)
	}
	p.mu.Unlock()
}

func (p *lru) Victim() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	oldest := p.order.Back()
	if oldest == nil {
		return "", false
	}
	return oldest.Value.(string), true
}

// touchLocked moves key to the front of the list, inserting it when it is not
// yet tracked. The caller must hold p.mu.
func (p *lru) touchLocked(key string) {
	if element, ok := p.entries[key]; ok {
		p.order.MoveToFront(element)
		return
	}
	p.entries[key] = p.order.PushFront(key)
}
