package storage

import (
	"errors"
	"hash/fnv"
	"math"
)

var ErrBloomFilterKeyNotFound = errors.New("bloom filter key does not exist")
var ErrBloomFilterIncompatible = errors.New("invalid bloom filter configuration")

const (
	defaultBloomFilterErrorRate = 0.01
	defaultBloomFilterCapacity  = 1000
)

type BloomFilterInfo struct {
	Capacity  uint64
	Size      uint64
	HashCount uint64
	Count     uint64
}

type BloomFilter struct {
	bits      []uint64
	bitCount  uint64
	hashCount uint64
	capacity  uint64
	count     uint64
}

// size approximates the bloom filter footprint from its bit array.
func (f *BloomFilter) size() int64 {
	return int64(len(f.bits)) * 8
}

func NewBloomFilter(errorRate float64, capacity uint64) (*BloomFilter, error) {
	if errorRate <= 0 || errorRate >= 1 || math.IsNaN(errorRate) || capacity == 0 {
		return nil, ErrBloomFilterIncompatible
	}

	bitCount := uint64(math.Ceil(-float64(capacity) * math.Log(errorRate) / (math.Ln2 * math.Ln2)))
	if bitCount == 0 || bitCount > math.MaxUint64-63 {
		return nil, ErrBloomFilterIncompatible
	}
	wordCount := (bitCount + 63) / 64
	if wordCount > uint64(^uint(0)>>1) {
		return nil, ErrBloomFilterIncompatible
	}
	hashCount := uint64(math.Round(float64(bitCount) / float64(capacity) * math.Ln2))
	if hashCount == 0 {
		hashCount = 1
	}

	return &BloomFilter{
		bits:      make([]uint64, int(wordCount)),
		bitCount:  bitCount,
		hashCount: hashCount,
		capacity:  capacity,
	}, nil
}

func (s *Store) BFReserve(key string, errorRate float64, capacity uint64) error {
	filter, err := NewBloomFilter(errorRate, capacity)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if record, ok := s.data[key]; ok {
		if !isExpired(record) {
			return ErrKeyExists
		}
		s.removeExpired(key, record)
	}
	next := Record{Type: ValueTypeBloomFilter, Value: filter}
	s.data[key] = next
	s.trackInsert(key, next)
	s.evictLocked()
	return nil
}

func (s *Store) BFAdd(key, item string) (bool, error) {
	return s.BFAddMany(key, []string{item})
}

func (s *Store) BFAddMany(key string, items []string) (bool, error) {
	results, err := s.BFMultiAdd(key, items)
	if err != nil {
		return false, err
	}
	return results[0], nil
}

func (s *Store) BFMultiAdd(key string, items []string) ([]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filter, err := s.bloomFilterForAddLocked(key)
	if err != nil {
		return nil, err
	}
	added := make([]bool, len(items))
	for index, item := range items {
		added[index] = filter.add(item)
	}
	return added, nil
}

func (s *Store) BFExists(key, item string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filter, found, err := s.bloomFilterLocked(key)
	if err != nil || !found {
		return false, err
	}
	s.notifyAccess(key)
	return filter.exists(item), nil
}

func (s *Store) BFMultiExists(key string, items []string) ([]bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filter, found, err := s.bloomFilterLocked(key)
	if err != nil {
		return nil, err
	}
	results := make([]bool, len(items))
	if !found {
		return results, nil
	}
	s.notifyAccess(key)
	for index, item := range items {
		results[index] = filter.exists(item)
	}
	return results, nil
}

func (s *Store) BFInfo(key string) (BloomFilterInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filter, found, err := s.bloomFilterLocked(key)
	if err != nil {
		return BloomFilterInfo{}, err
	}
	if !found {
		return BloomFilterInfo{}, ErrBloomFilterKeyNotFound
	}
	s.notifyAccess(key)
	return BloomFilterInfo{Capacity: filter.capacity, Size: uint64(len(filter.bits)) * 8, HashCount: filter.hashCount, Count: filter.count}, nil
}

func (s *Store) bloomFilterForAddLocked(key string) (*BloomFilter, error) {
	filter, found, err := s.bloomFilterLocked(key)
	if err != nil {
		return nil, err
	}
	if found {
		s.notifyAccess(key)
		return filter, nil
	}
	if record, ok := s.data[key]; ok {
		s.removeExpired(key, record)
	}
	filter, err = NewBloomFilter(defaultBloomFilterErrorRate, defaultBloomFilterCapacity)
	if err != nil {
		return nil, err
	}
	next := Record{Type: ValueTypeBloomFilter, Value: filter}
	s.data[key] = next
	s.trackInsert(key, next)
	s.evictLocked()
	return filter, nil
}

func (s *Store) bloomFilterLocked(key string) (*BloomFilter, bool, error) {
	record, ok := s.data[key]
	if !ok || isExpired(record) {
		return nil, false, nil
	}
	if record.Type != ValueTypeBloomFilter {
		return nil, false, ErrWrongType
	}
	filter, ok := record.Value.(*BloomFilter)
	if !ok {
		return nil, false, ErrWrongType
	}
	return filter, true, nil
}

func (f *BloomFilter) add(item string) bool {
	exists := f.exists(item)
	for hashIndex := uint64(0); hashIndex < f.hashCount; hashIndex++ {
		position := hashBloomItem(item, hashIndex) % f.bitCount
		f.bits[position/64] |= uint64(1) << (position % 64)
	}
	if !exists {
		f.count++
	}
	return !exists
}

func (f *BloomFilter) exists(item string) bool {
	for hashIndex := uint64(0); hashIndex < f.hashCount; hashIndex++ {
		position := hashBloomItem(item, hashIndex) % f.bitCount
		if f.bits[position/64]&(uint64(1)<<(position%64)) == 0 {
			return false
		}
	}
	return true
}

func hashBloomItem(item string, index uint64) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(item))
	var seed [8]byte
	for byteIndex := range seed {
		seed[byteIndex] = byte(index >> (byteIndex * 8))
	}
	_, _ = hasher.Write(seed[:])
	return hasher.Sum64()
}
