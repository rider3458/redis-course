package storage

import (
	"errors"
	"sync"
	"time"
)

var ErrNotInteger = errors.New("value is not an integer or out of range")
var ErrWrongType = errors.New("wrong type")

type ValueType uint8

const (
	ValueTypeString ValueType = iota
	ValueTypeSimpleSet
	ValueTypeSortedSet
	ValueTypeCountMinSketch
)

func (t ValueType) String() string {
	switch t {
	case ValueTypeString:
		return "string"
	case ValueTypeSimpleSet:
		return "set"
	case ValueTypeSortedSet:
		return "zset"
	case ValueTypeCountMinSketch:
		return "cms"
	default:
		return "unknown"
	}
}

// Record stores a value and TTL metadata for a key.
//
// TTL holds an absolute expiration timestamp in UTC.
// A zero value means the key does not expire.

type Record struct {
	Type  ValueType
	Value any
	TTL   time.Time
}

// Store is an in-memory, concurrency-safe key/value store.
type Store struct {
	mu            sync.RWMutex
	data          map[string]Record
	sortedSetType SortedSetType
}

// Option configures a Store.
type Option func(*Store)

// WithSortedSetType configures the underlying index implementation for new sorted sets.
func WithSortedSetType(t SortedSetType) Option {
	return func(s *Store) {
		s.sortedSetType = t
	}
}

// WithBPlusTree configures the store to use B+ tree for sorted sets.
func WithBPlusTree() Option {
	return func(s *Store) {
		s.sortedSetType = SortedSetTypeBPlusTree
	}
}

// WithSkipList configures the store to use SkipList for sorted sets.
func WithSkipList() Option {
	return func(s *Store) {
		s.sortedSetType = SortedSetTypeSkipList
	}
}

func New(opts ...Option) *Store {
	s := &Store{
		data:          make(map[string]Record),
		sortedSetType: SortedSetTypeSkipList,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Store) SetSortedSetType(t SortedSetType) {
	s.mu.Lock()
	s.sortedSetType = t
	s.mu.Unlock()
}

func (s *Store) SortedSetType() SortedSetType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sortedSetType
}
