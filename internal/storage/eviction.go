package storage

import "strings"

// evictionSampleSize is the number of tracked keys examined when selecting a
// victim for the sampled LFU policy.
const evictionSampleSize = 5

// EvictionPolicy tracks keys for a Store and selects victims to evict when the
// store exceeds its configured capacity.
//
// The Store invokes these methods while holding its internal lock, so
// implementations must not call back into the Store. Methods may be invoked
// concurrently by read operations, so implementations must be safe for
// concurrent use.
type EvictionPolicy interface {
	// Name returns the canonical policy name, such as "lru".
	Name() string
	// OnInsert records a newly inserted key.
	OnInsert(key string)
	// OnAccess records a read or write hit on an existing key.
	OnAccess(key string)
	// OnRemove drops tracking for a removed key.
	OnRemove(key string)
	// Victim returns the next key to evict. ok is false when no key is tracked.
	Victim() (key string, ok bool)
}

// WithMaxEntries evicts keys once the number of live keys exceeds n.
// A value of zero means no entry limit.
func WithMaxEntries(n int) Option {
	return func(s *Store) {
		s.maxEntries = n
	}
}

// WithMaxMemory evicts keys once approximate memory usage exceeds n bytes.
// A value of zero means no memory limit.
func WithMaxMemory(n int64) Option {
	return func(s *Store) {
		s.maxMemory = n
	}
}

// WithEvictionPolicy configures the policy used to select eviction victims.
// A nil policy disables eviction.
func WithEvictionPolicy(p EvictionPolicy) Option {
	return func(s *Store) {
		s.evictionPolicy = p
	}
}

// WithLRU configures exact least-recently-used eviction.
func WithLRU() Option {
	return WithEvictionPolicy(NewLRU())
}

// WithRandomEviction configures random eviction.
func WithRandomEviction() Option {
	return WithEvictionPolicy(NewRandomEviction())
}

// WithLFU configures least-frequently-used eviction.
func WithLFU() Option {
	return WithEvictionPolicy(NewLFU())
}

// EvictionPolicyByName resolves a configured policy name to an EvictionPolicy.
//
// It accepts "lru", "random", "lfu" and their Redis-style "allkeys-" aliases;
// "noeviction" (or "none") returns a nil policy. The second result is false
// for an unknown name.
func EvictionPolicyByName(name string) (EvictionPolicy, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "lru", "allkeys-lru":
		return NewLRU(), true
	case "random", "allkeys-random":
		return NewRandomEviction(), true
	case "lfu", "allkeys-lfu":
		return NewLFU(), true
	case "noeviction", "none":
		return nil, true
	default:
		return nil, false
	}
}

// Evict removes keys per the configured policy until the store is within
// capacity and returns the number of keys evicted.
func (s *Store) Evict() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.evictLocked()
}

// Sweep actively removes expired keys and then evicts until the store is within
// capacity. It returns the number of keys removed by each step.
//
// ponytail: O(n) scan under the store lock per sweep; move to an expiry index
// if sweeps show up in profiles.
func (s *Store) Sweep() (expired int, evicted int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, record := range s.data {
		if isExpired(record) {
			s.removeExpired(key, record)
			expired++
		}
	}
	return expired, s.evictLocked()
}

// evictLocked removes keys until the store is within capacity. The caller must
// hold s.mu.
func (s *Store) evictLocked() int {
	if s.evictionPolicy == nil {
		return 0
	}

	evicted := 0
	for s.overCapacity() {
		key, ok := s.evictionPolicy.Victim()
		if !ok {
			break
		}

		record, exists := s.data[key]
		if !exists {
			s.evictionPolicy.OnRemove(key)
			continue
		}

		delete(s.data, key)
		s.trackRemove(key, record)
		s.evictedKeys.Add(1)
		evicted++
	}
	return evicted
}

// overCapacity reports whether the store exceeds an entry or memory limit. The
// caller must hold s.mu.
func (s *Store) overCapacity() bool {
	if s.maxEntries > 0 && len(s.data) > s.maxEntries {
		return true
	}
	return s.maxMemory > 0 && s.usedMemory.Load() > s.maxMemory
}
