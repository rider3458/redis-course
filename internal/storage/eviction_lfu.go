package storage

import (
	"sync"
	"sync/atomic"
)

// lfuEntry tracks how often a key was used and when it was last used, so that
// frequency ties break toward the least recently used key.
type lfuEntry struct {
	freq int64
	last int64
}

// lfu implements EvictionPolicy with sampled least-frequently-used selection.
type lfu struct {
	mu      sync.Mutex
	entries map[string]lfuEntry
	seq     atomic.Int64
	sample  int
}

// NewLFU returns a least-frequently-used eviction policy.
func NewLFU() EvictionPolicy {
	return &lfu{
		entries: make(map[string]lfuEntry),
		sample:  evictionSampleSize,
	}
}

func (p *lfu) Name() string { return "lfu" }

func (p *lfu) OnInsert(key string) {
	p.mu.Lock()
	p.entries[key] = lfuEntry{freq: 1, last: p.seq.Add(1)}
	p.mu.Unlock()
}

func (p *lfu) OnAccess(key string) {
	p.mu.Lock()
	entry := p.entries[key]
	entry.freq++
	entry.last = p.seq.Add(1)
	p.entries[key] = entry
	p.mu.Unlock()
}

func (p *lfu) OnRemove(key string) {
	p.mu.Lock()
	delete(p.entries, key)
	p.mu.Unlock()
}

func (p *lfu) Victim() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var victim string
	var victimEntry lfuEntry
	found := false
	sampled := 0
	for key, entry := range p.entries {
		if !found || lessFrequent(entry, victimEntry) {
			victim, victimEntry, found = key, entry, true
		}
		sampled++
		if sampled >= p.sample {
			break
		}
	}
	return victim, found
}

// lessFrequent reports whether a should be evicted before b: lower frequency
// wins, and equal frequencies break toward the older access.
func lessFrequent(a, b lfuEntry) bool {
	if a.freq != b.freq {
		return a.freq < b.freq
	}
	return a.last < b.last
}
