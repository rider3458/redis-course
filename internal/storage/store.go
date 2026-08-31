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
)

func (t ValueType) String() string {
	switch t {
	case ValueTypeString:
		return "string"
	case ValueTypeSimpleSet:
		return "set"
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
	mu   sync.RWMutex
	data map[string]Record
}
