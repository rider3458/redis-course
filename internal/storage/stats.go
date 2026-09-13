package storage

// StoreStats reports counters that power the INFO command.
type StoreStats struct {
	Keys           int
	ExpiredKeys    int64
	EvictedKeys    int64
	Hits           int64
	Misses         int64
	UsedMemory     int64
	MaxMemory      int64
	EvictionPolicy string
}

// Stats returns a snapshot of the store's current statistics.
func (s *Store) Stats() StoreStats {
	s.mu.RLock()
	keys := len(s.data)
	maxMemory := s.maxMemory
	policy := s.evictionPolicy
	s.mu.RUnlock()

	name := "noeviction"
	if policy != nil {
		name = policy.Name()
	}

	return StoreStats{
		Keys:           keys,
		ExpiredKeys:    s.expiredKeys.Load(),
		EvictedKeys:    s.evictedKeys.Load(),
		Hits:           s.hits.Load(),
		Misses:         s.misses.Load(),
		UsedMemory:     s.usedMemory.Load(),
		MaxMemory:      maxMemory,
		EvictionPolicy: name,
	}
}

// valueSize returns the approximate payload footprint of a record value.
func valueSize(value any) int64 {
	switch typed := value.(type) {
	case string:
		return int64(len(typed))
	case map[string]struct{}:
		var total int64
		for member := range typed {
			total += int64(len(member))
		}
		return total
	case *SortedSet:
		return typed.size()
	case *CountMinSketch:
		return typed.size()
	case *BloomFilter:
		return typed.size()
	default:
		return 0
	}
}

// recordSize returns the approximate memory footprint of a key and value.
func recordSize(key string, record Record) int64 {
	return int64(len(key)) + valueSize(record.Value)
}

// accountMemory adjusts used memory by delta. The caller must hold s.mu.
func (s *Store) accountMemory(delta int64) {
	if delta != 0 {
		s.usedMemory.Add(delta)
	}
}

// notifyInsert informs the eviction policy that key was created. The caller
// must hold s.mu.
func (s *Store) notifyInsert(key string) {
	if s.evictionPolicy != nil {
		s.evictionPolicy.OnInsert(key)
	}
}

// notifyAccess informs the eviction policy that a live key was accessed. The
// caller must hold s.mu.
func (s *Store) notifyAccess(key string) {
	if s.evictionPolicy != nil {
		s.evictionPolicy.OnAccess(key)
	}
}

// notifyRemove informs the eviction policy that key was removed. The caller
// must hold s.mu.
func (s *Store) notifyRemove(key string) {
	if s.evictionPolicy != nil {
		s.evictionPolicy.OnRemove(key)
	}
}

// trackInsert records a newly created key. The caller must hold s.mu.
func (s *Store) trackInsert(key string, record Record) {
	s.usedMemory.Add(recordSize(key, record))
	s.notifyInsert(key)
}

// trackOverwrite records a write to an existing live key. The caller must hold
// s.mu.
func (s *Store) trackOverwrite(key string, previous, next Record) {
	s.usedMemory.Add(recordSize(key, next) - recordSize(key, previous))
	s.notifyAccess(key)
}

// trackRemove records the removal of a key. The caller must hold s.mu.
func (s *Store) trackRemove(key string, record Record) {
	s.usedMemory.Add(-recordSize(key, record))
	s.notifyRemove(key)
}

// removeExpired deletes an already-expired key and records the expiry. The
// caller must hold s.mu.
func (s *Store) removeExpired(key string, record Record) {
	delete(s.data, key)
	s.expiredKeys.Add(1)
	s.trackRemove(key, record)
}
