package storage

import (
	"errors"
	"strconv"
	"sync"
	"time"
)

var ErrNotInteger = errors.New("value is not an integer or out of range")

// Record stores a value and TTL metadata for a key.
//
// TTL holds an absolute expiration timestamp in UTC.
// A zero value means the key does not expire.
type Record struct {
	Value string
	TTL   time.Time
}

// Store is an in-memory, concurrency-safe key/value store.
type Store struct {
	mu   sync.RWMutex
	data map[string]Record
}

func New() *Store {
	return &Store{
		data: make(map[string]Record),
	}
}

func (s *Store) Set(key string, value string, ttl time.Duration) {
	record := Record{Value: value}
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
	s.mu.RUnlock()
	if !ok {
		return "", false
	}

	if isExpired(record) {
		s.mu.Lock()
		current, exists := s.data[key]
		if exists && isExpired(current) {
			delete(s.data, key)
		}
		s.mu.Unlock()
		return "", false
	}

	return record.Value, true
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
	s.mu.RUnlock()
	if !ok {
		return -2
	}

	if isExpired(record) {
		s.mu.Lock()
		current, exists := s.data[key]
		if exists && isExpired(current) {
			delete(s.data, key)
		}
		s.mu.Unlock()
		return -2
	}

	if record.TTL.IsZero() {
		return -1
	}

	remaining := time.Until(record.TTL)
	if remaining <= 0 {
		s.mu.Lock()
		current, exists := s.data[key]
		if exists && isExpired(current) {
			delete(s.data, key)
		}
		s.mu.Unlock()
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
		if _, ok := s.Get(key); ok {
			count++
		}
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
		parsed, err := strconv.ParseInt(record.Value, 10, 64)
		if err != nil {
			return 0, ErrNotInteger
		}
		current = parsed
		ttl = record.TTL
	}

	next := current + 1
	s.data[key] = Record{Value: strconv.FormatInt(next, 10), TTL: ttl}
	return next, nil
}

func isExpired(record Record) bool {
	if record.TTL.IsZero() {
		return false
	}
	return time.Now().UTC().After(record.TTL)
}
