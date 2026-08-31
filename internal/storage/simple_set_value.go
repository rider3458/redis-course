package storage

import "sort"

func (s *Store) SAdd(key string, members ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data[key]
	if ok && isExpired(record) {
		delete(s.data, key)
		ok = false
	}

	if !ok {
		set := make(map[string]struct{}, len(members))
		added := 0
		for _, member := range members {
			if _, exists := set[member]; exists {
				continue
			}
			set[member] = struct{}{}
			added++
		}
		s.data[key] = Record{Type: ValueTypeSimpleSet, Value: set}
		return added, nil
	}

	if record.Type != ValueTypeSimpleSet {
		return 0, ErrWrongType
	}

	set, ok := record.Value.(map[string]struct{})
	if !ok {
		return 0, ErrWrongType
	}

	added := 0
	for _, member := range members {
		if _, exists := set[member]; exists {
			continue
		}
		set[member] = struct{}{}
		added++
	}

	record.Value = set
	s.data[key] = record
	return added, nil
}

func (s *Store) SRem(key string, members ...string) (int, error) {
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
	if record.Type != ValueTypeSimpleSet {
		return 0, ErrWrongType
	}

	set, ok := record.Value.(map[string]struct{})
	if !ok {
		return 0, ErrWrongType
	}

	removed := 0
	for _, member := range members {
		if _, exists := set[member]; !exists {
			continue
		}
		delete(set, member)
		removed++
	}

	if len(set) == 0 {
		delete(s.data, key)
		return removed, nil
	}

	record.Value = set
	s.data[key] = record
	return removed, nil
}

func (s *Store) SIsMember(key string, member string) (int, error) {
	s.mu.RLock()
	record, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return 0, nil
	}
	if isExpired(record) {
		s.mu.Lock()
		current, exists := s.data[key]
		if exists && isExpired(current) {
			delete(s.data, key)
		}
		s.mu.Unlock()
		return 0, nil
	}
	if record.Type != ValueTypeSimpleSet {
		return 0, ErrWrongType
	}

	set, ok := record.Value.(map[string]struct{})
	if !ok {
		return 0, ErrWrongType
	}

	if _, exists := set[member]; exists {
		return 1, nil
	}
	return 0, nil
}

func (s *Store) SMembers(key string) ([]string, error) {
	s.mu.RLock()
	record, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return []string{}, nil
	}
	if isExpired(record) {
		s.mu.Lock()
		current, exists := s.data[key]
		if exists && isExpired(current) {
			delete(s.data, key)
		}
		s.mu.Unlock()
		return []string{}, nil
	}
	if record.Type != ValueTypeSimpleSet {
		return nil, ErrWrongType
	}

	set, ok := record.Value.(map[string]struct{})
	if !ok {
		return nil, ErrWrongType
	}

	members := make([]string, 0, len(set))
	for member := range set {
		members = append(members, member)
	}
	sort.Strings(members)
	return members, nil
}
