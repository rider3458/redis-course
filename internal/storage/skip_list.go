package storage

import "math/rand/v2"

const maxSkipListHeight = 32

type SkipListNode struct {
	member string
	score  float64
	next   [maxSkipListHeight]*SkipListNode
}

type SkipList struct {
	head    *SkipListNode
	members map[string]*SkipListNode
	length  int
}

func NewSkipList() *SkipList {
	return &SkipList{
		head:    &SkipListNode{},
		members: make(map[string]*SkipListNode),
	}
}

func (s *SkipList) add(member string, score float64) bool {
	if node, ok := s.members[member]; ok {
		if node.score != score {
			s.update(member, score)
		}
		return false
	}

	update := s.predecessors(score, member)
	newNode := &SkipListNode{member: member, score: score}
	for level := 0; level < s.randomHeight(); level++ {
		newNode.next[level] = update[level].next[level]
		update[level].next[level] = newNode
	}
	s.members[member] = newNode
	s.length++
	return true
}

func (s *SkipList) update(member string, score float64) bool {
	node, ok := s.members[member]
	if !ok {
		return false
	}
	if node.score == score {
		return true
	}

	s.remove(member)
	s.add(member, score)
	return true
}

func (s *SkipList) remove(member string) bool {
	node, ok := s.members[member]
	if !ok {
		return false
	}

	update := s.predecessors(node.score, member)
	for level := range maxSkipListHeight {
		if update[level].next[level] == node {
			update[level].next[level] = node.next[level]
		}
	}
	delete(s.members, member)
	s.length--
	return true
}

func (s *SkipList) getMemberScore(member string) (float64, bool) {
	node, ok := s.members[member]
	if !ok {
		return 0, false
	}
	return node.score, true
}

func (s *SkipList) getRank(member string) (int, bool) {
	rank := 0
	for node := s.head.next[0]; node != nil; node = node.next[0] {
		if node.member == member {
			return rank, true
		}
		rank++
	}
	return 0, false
}

func (s *SkipList) getRange(start, stop int) []string {
	if start < 0 {
		start += s.length
	}
	if stop < 0 {
		stop += s.length
	}
	if start < 0 {
		start = 0
	}
	if stop >= s.length {
		stop = s.length - 1
	}
	if start > stop || start >= s.length {
		return []string{}
	}

	members := make([]string, 0, stop-start+1)
	index := 0
	for node := s.head.next[0]; node != nil && index <= stop; node = node.next[0] {
		if index >= start {
			members = append(members, node.member)
		}
		index++
	}
	return members
}

func (s *SkipList) isMember(member string) bool {
	_, ok := s.members[member]
	return ok
}

func (s *SkipList) len() int {
	return s.length
}

func (s *SkipList) Add(member string, score float64) bool {
	return s.add(member, score)
}

func (s *SkipList) Update(member string, score float64) bool {
	return s.update(member, score)
}

func (s *SkipList) Remove(member string) bool {
	return s.remove(member)
}

func (s *SkipList) GetMemberScore(member string) (float64, bool) {
	return s.getMemberScore(member)
}

func (s *SkipList) GetRank(member string) (int, bool) {
	return s.getRank(member)
}

func (s *SkipList) GetRange(start, stop int) []string {
	return s.getRange(start, stop)
}

func (s *SkipList) IsMember(member string) bool {
	return s.isMember(member)
}

func (s *SkipList) randomHeight() int {
	height := 1
	for height < maxSkipListHeight && rand.IntN(2) == 1 {
		height++
	}
	return height
}

func (s *SkipList) predecessors(score float64, member string) [maxSkipListHeight]*SkipListNode {
	var update [maxSkipListHeight]*SkipListNode
	node := s.head
	for level := maxSkipListHeight - 1; level >= 0; level-- {
		for next := node.next[level]; next != nil && less(next.score, next.member, score, member); next = node.next[level] {
			node = next
		}
		update[level] = node
	}
	return update
}

func less(leftScore float64, leftMember string, rightScore float64, rightMember string) bool {
	return leftScore < rightScore || leftScore == rightScore && leftMember < rightMember
}
