package storage

import (
	"strconv"
	"time"
)

func (s *Store) Set(key string, value string, ttl time.Duration) {
	record := Record{Type: ValueTypeString, Value: value}
	if ttl > 0 {
		record.TTL = time.Now().UTC().Add(ttl)
	}

	s.mu.Lock()
	if previous, ok := s.data[key]; ok {
		if isExpired(previous) {
			s.removeExpired(key, previous)
		} else {
			s.data[key] = record
			s.trackOverwrite(key, previous, record)
			s.evictLocked()
			s.mu.Unlock()
			return
		}
	}
	s.data[key] = record
	s.trackInsert(key, record)
	s.evictLocked()
	s.mu.Unlock()
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	record, ok := s.data[key]
	if !ok {
		s.mu.RUnlock()
		s.misses.Add(1)
		return "", false
	}

	if isExpired(record) {
		s.mu.RUnlock()
		s.deleteIfExpired(key)
		s.misses.Add(1)
		return "", false
	}

	value, ok := record.Value.(string)
	if !ok {
		s.mu.RUnlock()
		s.misses.Add(1)
		return "", false
	}

	s.hits.Add(1)
	s.notifyAccess(key)
	s.mu.RUnlock()

	return value, true
}

func (s *Store) Expire(key string, ttl time.Duration) bool {
	if ttl <= 0 {
		return s.Delete(key) > 0
	}

	expiresAt := time.Now().UTC().Add(ttl)

	s.mu.Lock()
	record, ok := s.data[key]
	if !ok {
		s.mu.Unlock()
		return false
	}
	if isExpired(record) {
		s.removeExpired(key, record)
		s.mu.Unlock()
		return false
	}
	record.TTL = expiresAt
	s.data[key] = record
	s.notifyAccess(key)
	s.mu.Unlock()
	return true
}

func (s *Store) TTL(key string) int64 {
	s.mu.RLock()
	record, ok := s.data[key]
	if !ok {
		s.mu.RUnlock()
		return -2
	}

	if isExpired(record) {
		s.mu.RUnlock()
		s.deleteIfExpired(key)
		return -2
	}

	if record.TTL.IsZero() {
		s.mu.RUnlock()
		return -1
	}

	remaining := time.Until(record.TTL)
	s.mu.RUnlock()
	if remaining <= 0 {
		s.deleteIfExpired(key)
		return -2
	}

	return int64(remaining / time.Second)
}

func (s *Store) Delete(keys ...string) int {
	deleted := 0

	s.mu.Lock()
	for _, key := range keys {
		if record, ok := s.data[key]; ok {
			delete(s.data, key)
			s.trackRemove(key, record)
			deleted++
		}
	}
	s.mu.Unlock()

	return deleted
}

func (s *Store) Exists(keys ...string) int {
	count := 0
	for _, key := range keys {
		s.mu.RLock()
		record, ok := s.data[key]
		if !ok {
			s.mu.RUnlock()
			continue
		}

		if isExpired(record) {
			s.mu.RUnlock()
			s.deleteIfExpired(key)
			continue
		}
		s.mu.RUnlock()

		count++
	}
	return count
}

func (s *Store) Incr(key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data[key]
	if ok && isExpired(record) {
		s.removeExpired(key, record)
		ok = false
	}

	var current int64
	var ttl time.Time
	if ok {
		value, valueOK := record.Value.(string)
		if !valueOK {
			return 0, ErrNotInteger
		}

		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, ErrNotInteger
		}
		current = parsed
		ttl = record.TTL
	}

	next := current + 1
	nextRecord := Record{Type: ValueTypeString, Value: strconv.FormatInt(next, 10), TTL: ttl}
	s.data[key] = nextRecord
	if ok {
		s.trackOverwrite(key, record, nextRecord)
	} else {
		s.trackInsert(key, nextRecord)
	}
	s.evictLocked()
	return next, nil
}

func isExpired(record Record) bool {
	if record.TTL.IsZero() {
		return false
	}
	return time.Now().UTC().After(record.TTL)
}

func (s *Store) deleteIfExpired(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record, ok := s.data[key]; ok && isExpired(record) {
		s.removeExpired(key, record)
	}
}
