package storage

type SortedSet struct {
	list *SkipList
}

func (s *Store) ZAdd(key string, members map[string]float64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data[key]
	if ok && isExpired(record) {
		delete(s.data, key)
		ok = false
	}

	if !ok {
		set := &SortedSet{list: NewSkipList()}
		added := 0
		for member, score := range members {
			if set.list.add(member, score) {
				added++
			}
		}
		s.data[key] = Record{Type: ValueTypeSortedSet, Value: set}
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
	for member, score := range members {
		if set.list.add(member, score) {
			added++
		}
	}
	return added, nil
}

func (s *Store) ZScore(key, member string) (float64, bool, error) {
	set, found, err := s.sortedSet(key)
	if err != nil || !found {
		return 0, false, err
	}
	score, found := set.list.getMemberScore(member)
	return score, found, nil
}

func (s *Store) ZRank(key, member string) (int, bool, error) {
	set, found, err := s.sortedSet(key)
	if err != nil || !found {
		return 0, false, err
	}
	rank, found := set.list.getRank(member)
	return rank, found, nil
}

func (s *Store) ZRange(key string, start, stop int) ([]string, error) {
	set, found, err := s.sortedSet(key)
	if err != nil {
		return nil, err
	}
	if !found {
		return []string{}, nil
	}
	return set.list.getRange(start, stop), nil
}

func (s *Store) ZRem(key string, members ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data[key]
	if !ok {
		return 0, nil
	}
	if isExpired(record) {
		delete(s.data, key)
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
	for _, member := range members {
		if set.list.remove(member) {
			removed++
		}
	}
	if set.list.length == 0 {
		delete(s.data, key)
	}
	return removed, nil
}

func (s *Store) sortedSet(key string) (*SortedSet, bool, error) {
	s.mu.RLock()
	record, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if isExpired(record) {
		s.mu.Lock()
		current, exists := s.data[key]
		if exists && isExpired(current) {
			delete(s.data, key)
		}
		s.mu.Unlock()
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
