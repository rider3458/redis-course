package storage

import "sort"

func (s *Store) SAdd(key string, members ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data[key]
	if ok && isExpired(record) {
		s.removeExpired(key, record)
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
		next := Record{Type: ValueTypeSimpleSet, Value: set}
		s.data[key] = next
		s.trackInsert(key, next)
		s.evictLocked()
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
	var memory int64
	for _, member := range members {
		if _, exists := set[member]; exists {
			continue
		}
		set[member] = struct{}{}
		added++
		memory += int64(len(member))
	}

	record.Value = set
	s.data[key] = record
	s.accountMemory(memory)
	s.notifyAccess(key)
	s.evictLocked()
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
		s.removeExpired(key, record)
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
	var memory int64
	for _, member := range members {
		if _, exists := set[member]; !exists {
			continue
		}
		delete(set, member)
		removed++
		memory -= int64(len(member))
	}
	s.accountMemory(memory)

	if len(set) == 0 {
		delete(s.data, key)
		s.trackRemove(key, record)
		return removed, nil
	}

	record.Value = set
	s.data[key] = record
	s.notifyAccess(key)
	return removed, nil
}

func (s *Store) SIsMember(key string, member string) (int, error) {
	s.mu.RLock()
	record, ok := s.data[key]
	if !ok {
		s.mu.RUnlock()
		return 0, nil
	}
	if isExpired(record) {
		s.mu.RUnlock()
		s.deleteIfExpired(key)
		return 0, nil
	}
	if record.Type != ValueTypeSimpleSet {
		s.mu.RUnlock()
		return 0, ErrWrongType
	}

	set, ok := record.Value.(map[string]struct{})
	if !ok {
		s.mu.RUnlock()
		return 0, ErrWrongType
	}

	_, exists := set[member]
	s.notifyAccess(key)
	s.mu.RUnlock()
	if exists {
		return 1, nil
	}
	return 0, nil
}

func (s *Store) SMembers(key string) ([]string, error) {
	s.mu.RLock()
	record, ok := s.data[key]
	if !ok {
		s.mu.RUnlock()
		return []string{}, nil
	}
	if isExpired(record) {
		s.mu.RUnlock()
		s.deleteIfExpired(key)
		return []string{}, nil
	}
	if record.Type != ValueTypeSimpleSet {
		s.mu.RUnlock()
		return nil, ErrWrongType
	}

	set, ok := record.Value.(map[string]struct{})
	if !ok {
		s.mu.RUnlock()
		return nil, ErrWrongType
	}

	members := make([]string, 0, len(set))
	for member := range set {
		members = append(members, member)
	}
	s.notifyAccess(key)
	s.mu.RUnlock()
	sort.Strings(members)
	return members, nil
}
