package storage

import (
	"errors"
	"hash/fnv"
	"math"
)

var ErrKeyExists = errors.New("key already exists")
var ErrCMSKeyNotFound = errors.New("count-min sketch key does not exist")
var ErrCMSIncompatible = errors.New("count-min sketches are incompatible")
var ErrCMSOverflow = errors.New("count-min sketch counter overflow")

type CMSIncrement struct {
	Item  string
	Value uint64
}

type CMSInfo struct {
	Width uint64
	Depth uint64
	Count uint64
}

type CountMinSketch struct {
	width    uint64
	depth    uint64
	counters [][]uint64
	count    uint64
}

// size approximates the sketch footprint from its counter matrix.
func (s *CountMinSketch) size() int64 {
	return int64(s.width) * int64(s.depth) * 8
}

func NewCountMinSketch(width, depth uint64) (*CountMinSketch, error) {
	if width == 0 || depth == 0 || width > uint64(^uint(0)>>1) || depth > uint64(^uint(0)>>1) {
		return nil, ErrCMSIncompatible
	}

	counters := make([][]uint64, depth)
	for row := range counters {
		counters[row] = make([]uint64, width)
	}

	return &CountMinSketch{width: width, depth: depth, counters: counters}, nil
}

func NewCountMinSketchByProbability(errorRate, probability float64) (*CountMinSketch, error) {
	if errorRate <= 0 || errorRate >= 1 || probability <= 0 || probability >= 1 || math.IsNaN(errorRate) || math.IsNaN(probability) {
		return nil, ErrCMSIncompatible
	}

	width := uint64(math.Ceil(math.E / errorRate))
	depth := uint64(math.Ceil(math.Log(1 / probability)))
	return NewCountMinSketch(width, depth)
}

func (s *Store) CMSInit(key string, width, depth uint64) error {
	sketch, err := NewCountMinSketch(width, depth)
	if err != nil {
		return err
	}
	return s.cmsInitSketch(key, sketch)
}

func (s *Store) CMSInitByProbability(key string, errorRate, probability float64) error {
	sketch, err := NewCountMinSketchByProbability(errorRate, probability)
	if err != nil {
		return err
	}
	return s.cmsInitSketch(key, sketch)
}

func (s *Store) cmsInitSketch(key string, sketch *CountMinSketch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record, ok := s.data[key]; ok {
		if !isExpired(record) {
			return ErrKeyExists
		}
		s.removeExpired(key, record)
	}
	next := Record{Type: ValueTypeCountMinSketch, Value: sketch}
	s.data[key] = next
	s.trackInsert(key, next)
	s.evictLocked()
	return nil
}

func (s *Store) CMSIncrBy(key string, increments []CMSIncrement) ([]uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sketch, err := s.countMinSketchLocked(key)
	if err != nil {
		return nil, err
	}

	results := make([]uint64, len(increments))
	for index, increment := range increments {
		if math.MaxUint64-sketch.count < increment.Value {
			return nil, ErrCMSOverflow
		}
		for row := uint64(0); row < sketch.depth; row++ {
			column := hashItem(increment.Item, row) % sketch.width
			if math.MaxUint64-sketch.counters[row][column] < increment.Value {
				return nil, ErrCMSOverflow
			}
		}
		for row := uint64(0); row < sketch.depth; row++ {
			column := hashItem(increment.Item, row) % sketch.width
			sketch.counters[row][column] += increment.Value
		}
		sketch.count += increment.Value
		results[index] = sketch.query(increment.Item)
	}

	s.notifyAccess(key)
	return results, nil
}

func (s *Store) CMSQuery(key string, items []string) ([]uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sketch, err := s.countMinSketchLocked(key)
	if err != nil {
		return nil, err
	}

	s.notifyAccess(key)
	counts := make([]uint64, len(items))
	for index, item := range items {
		counts[index] = sketch.query(item)
	}
	return counts, nil
}

func (s *Store) CMSInfo(key string) (CMSInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sketch, err := s.countMinSketchLocked(key)
	if err != nil {
		return CMSInfo{}, err
	}
	s.notifyAccess(key)
	return CMSInfo{Width: sketch.width, Depth: sketch.depth, Count: sketch.count}, nil
}

func (s *Store) CMSMerge(destination string, sourceKeys []string, weights []uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(sourceKeys) == 0 || len(sourceKeys) != len(weights) {
		return ErrCMSIncompatible
	}

	sources := make([]*CountMinSketch, len(sourceKeys))
	for index, key := range sourceKeys {
		sketch, err := s.countMinSketchLocked(key)
		if err != nil {
			return err
		}
		s.notifyAccess(key)
		sources[index] = sketch
	}

	width, depth := sources[0].width, sources[0].depth
	merged, err := NewCountMinSketch(width, depth)
	if err != nil {
		return err
	}
	for index, source := range sources {
		if source.width != width || source.depth != depth {
			return ErrCMSIncompatible
		}
		for row := uint64(0); row < depth; row++ {
			for column := uint64(0); column < width; column++ {
				if source.counters[row][column] != 0 && weights[index] > math.MaxUint64/source.counters[row][column] {
					return ErrCMSOverflow
				}
				addition := source.counters[row][column] * weights[index]
				if math.MaxUint64-merged.counters[row][column] < addition {
					return ErrCMSOverflow
				}
				merged.counters[row][column] += addition
			}
		}
		if source.count != 0 && weights[index] > math.MaxUint64/source.count {
			return ErrCMSOverflow
		}
		addition := source.count * weights[index]
		if math.MaxUint64-merged.count < addition {
			return ErrCMSOverflow
		}
		merged.count += addition
	}

	if record, ok := s.data[destination]; ok {
		if !isExpired(record) {
			if record.Type != ValueTypeCountMinSketch {
				return ErrWrongType
			}
			next := Record{Type: ValueTypeCountMinSketch, Value: merged}
			s.data[destination] = next
			s.trackOverwrite(destination, record, next)
			s.evictLocked()
			return nil
		}
		s.removeExpired(destination, record)
	}
	next := Record{Type: ValueTypeCountMinSketch, Value: merged}
	s.data[destination] = next
	s.trackInsert(destination, next)
	s.evictLocked()
	return nil
}

func (s *Store) countMinSketchLocked(key string) (*CountMinSketch, error) {
	record, ok := s.data[key]
	if !ok || isExpired(record) {
		return nil, ErrCMSKeyNotFound
	}
	if record.Type != ValueTypeCountMinSketch {
		return nil, ErrWrongType
	}
	sketch, ok := record.Value.(*CountMinSketch)
	if !ok {
		return nil, ErrWrongType
	}
	return sketch, nil
}

func (s *CountMinSketch) query(item string) uint64 {
	minimum := uint64(math.MaxUint64)
	for row := uint64(0); row < s.depth; row++ {
		column := hashItem(item, row) % s.width
		minimum = min(minimum, s.counters[row][column])
	}
	return minimum
}

func hashItem(item string, row uint64) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(item))
	var seed [8]byte
	for index := range seed {
		seed[index] = byte(row >> (index * 8))
	}
	_, _ = hasher.Write(seed[:])
	return hasher.Sum64()
}
