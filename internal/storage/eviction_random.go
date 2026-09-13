package storage

import "sync"

// randomEviction implements EvictionPolicy by tracking the set of live keys and
// returning an arbitrary victim on each selection.
type randomEviction struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

// NewRandomEviction returns a random eviction policy.
func NewRandomEviction() EvictionPolicy {
	return &randomEviction{
		keys: make(map[string]struct{}),
	}
}

func (p *randomEviction) Name() string { return "random" }

func (p *randomEviction) OnInsert(key string) {
	p.mu.Lock()
	p.keys[key] = struct{}{}
	p.mu.Unlock()
}

func (p *randomEviction) OnAccess(key string) {
	p.mu.Lock()
	p.keys[key] = struct{}{}
	p.mu.Unlock()
}

func (p *randomEviction) OnRemove(key string) {
	p.mu.Lock()
	delete(p.keys, key)
	p.mu.Unlock()
}

func (p *randomEviction) Victim() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Go randomizes map iteration order, so the first key is an arbitrary pick.
	for key := range p.keys {
		return key, true
	}
	return "", false
}
