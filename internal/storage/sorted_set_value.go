package storage

type SortedSetType int

const (
	SortedSetTypeSkipList SortedSetType = iota
	SortedSetTypeBPlusTree
)

const (
	SortedSetSkipList  = SortedSetTypeSkipList
	SortedSetBPlusTree = SortedSetTypeBPlusTree
)

type sortedSetIndex interface {
	add(member string, score float64) bool
	update(member string, score float64) bool
	remove(member string) bool
	getMemberScore(member string) (float64, bool)
	getRank(member string) (int, bool)
	getRange(start, stop int) []string
	isMember(member string) bool
	len() int
}

type SortedSet struct {
	index sortedSetIndex
}

func NewSortedSet(impl ...SortedSetType) *SortedSet {
	t := SortedSetTypeSkipList
	if len(impl) > 0 {
		t = impl[0]
	}
	switch t {
	case SortedSetTypeBPlusTree:
		return &SortedSet{index: NewBPlusTree()}
	default:
		return &SortedSet{index: NewSkipList()}
	}
}

func NewSortedSetWithSkipList() *SortedSet {
	return &SortedSet{index: NewSkipList()}
}

func NewSortedSetWithBPlusTree() *SortedSet {
	return &SortedSet{index: NewBPlusTree()}
}

func (s *SortedSet) Index() any {
	return s.index
}

func (s *SortedSet) Add(member string, score float64) bool {
	return s.index.add(member, score)
}

func (s *SortedSet) Update(member string, score float64) bool {
	return s.index.update(member, score)
}

func (s *SortedSet) Remove(member string) bool {
	return s.index.remove(member)
}

func (s *SortedSet) GetMemberScore(member string) (float64, bool) {
	return s.index.getMemberScore(member)
}

func (s *SortedSet) GetRank(member string) (int, bool) {
	return s.index.getRank(member)
}

func (s *SortedSet) GetRange(start, stop int) []string {
	return s.index.getRange(start, stop)
}

func (s *SortedSet) IsMember(member string) bool {
	return s.index.isMember(member)
}

func (s *SortedSet) Len() int {
	return s.index.len()
}

// sortedSetEntryOverhead approximates per-member bookkeeping beyond the member
// name itself, such as the score and node pointers.
const sortedSetEntryOverhead int64 = 8

func sortedSetMemberSize(member string) int64 {
	return int64(len(member)) + sortedSetEntryOverhead
}

func (s *SortedSet) size() int64 {
	if s.index.len() == 0 {
		return 0
	}
	var total int64
	for _, member := range s.index.getRange(0, s.index.len()-1) {
		total += sortedSetMemberSize(member)
	}
	return total
}

func (s *Store) ZAdd(key string, members map[string]float64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data[key]
	if ok && isExpired(record) {
		s.removeExpired(key, record)
		ok = false
	}

	if !ok {
		set := NewSortedSet(s.sortedSetType)
		added := 0
		for member, score := range members {
			if set.index.add(member, score) {
				added++
			}
		}
		next := Record{Type: ValueTypeSortedSet, Value: set}
		s.data[key] = next
		s.trackInsert(key, next)
		s.evictLocked()
		return added, nil
	}

	if record.Type != ValueTypeSortedSet {
		return 0, ErrWrongType
	}
	set, ok := record.Value.(*SortedSet)
	if !ok {
		return 0, ErrWrongType
	}

	added := 0
	var memory int64
	for member, score := range members {
		if set.index.add(member, score) {
			added++
			memory += sortedSetMemberSize(member)
		}
	}
	s.accountMemory(memory)
	s.notifyAccess(key)
	return added, nil
}

func (s *Store) ZScore(key, member string) (float64, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set, found, err := s.sortedSetLocked(key)
	if err != nil || !found {
		return 0, false, err
	}
	s.notifyAccess(key)
	score, found := set.index.getMemberScore(member)
	return score, found, nil
}

func (s *Store) ZRank(key, member string) (int, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set, found, err := s.sortedSetLocked(key)
	if err != nil || !found {
		return 0, false, err
	}
	s.notifyAccess(key)
	rank, found := set.index.getRank(member)
	return rank, found, nil
}

func (s *Store) ZRange(key string, start, stop int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set, found, err := s.sortedSetLocked(key)
	if err != nil {
		return nil, err
	}
	if !found {
		return []string{}, nil
	}
	s.notifyAccess(key)
	return set.index.getRange(start, stop), nil
}

func (s *Store) ZRem(key string, members ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data[key]
	if !ok {
		return 0, nil
	}
	if isExpired(record) {
		s.removeExpired(key, record)
		return 0, nil
	}
	if record.Type != ValueTypeSortedSet {
		return 0, ErrWrongType
	}
	set, ok := record.Value.(*SortedSet)
	if !ok {
		return 0, ErrWrongType
	}

	removed := 0
	var memory int64
	for _, member := range members {
		if set.index.remove(member) {
			removed++
			memory -= sortedSetMemberSize(member)
		}
	}
	s.accountMemory(memory)
	if set.index.len() == 0 {
		delete(s.data, key)
		s.trackRemove(key, record)
	} else {
		s.notifyAccess(key)
	}
	return removed, nil
}

func (s *Store) sortedSetLocked(key string) (*SortedSet, bool, error) {
	record, ok := s.data[key]
	if !ok {
		return nil, false, nil
	}
	if isExpired(record) {
		return nil, false, nil
	}
	if record.Type != ValueTypeSortedSet {
		return nil, false, ErrWrongType
	}
	set, ok := record.Value.(*SortedSet)
	if !ok {
		return nil, false, ErrWrongType
	}
	return set, true, nil
}
