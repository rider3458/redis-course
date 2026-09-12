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
	s.data[key] = record
	s.mu.Unlock()
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	record, ok := s.data[key]
	if !ok {
		s.mu.RUnlock()
		return "", false
	}

	if isExpired(record) {
		s.mu.RUnlock()
		s.deleteIfExpired(key)
		return "", false
	}

	value, ok := record.Value.(string)
	s.mu.RUnlock()
	if !ok {
		return "", false
	}

	return value, true
}

func (s *Store) Expire(key string, ttl time.Duration) bool {
	if ttl <= 0 {
		return s.Delete(key) > 0
	}

	expiresAt := time.Now().UTC().Add(ttl)

	s.mu.Lock()
	record, ok := s.data[key]
	if !ok || isExpired(record) {
		if ok && isExpired(record) {
			delete(s.data, key)
		}
		s.mu.Unlock()
		return false
	}
	record.TTL = expiresAt
	s.data[key] = record
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
		if _, ok := s.data[key]; ok {
			delete(s.data, key)
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
		delete(s.data, key)
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
	s.data[key] = Record{Type: ValueTypeString, Value: strconv.FormatInt(next, 10), TTL: ttl}
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
		delete(s.data, key)
	}
}
