package storage

import "sort"

const defaultBPlusTreeOrder = 32

type bPlusItem struct {
	score  float64
	member string
}

func bPlusLess(leftScore float64, leftMember string, rightScore float64, rightMember string) bool {
	return leftScore < rightScore || (leftScore == rightScore && leftMember < rightMember)
}

type BPlusTreeNode struct {
	isLeaf   bool
	items    []bPlusItem
	keys     []bPlusItem
	children []*BPlusTreeNode
	next     *BPlusTreeNode
	prev     *BPlusTreeNode
}

type BPlusTree struct {
	root     *BPlusTreeNode
	leafHead *BPlusTreeNode
	members  map[string]float64
	length   int
	order    int
}

type BTreePlus = BPlusTree

func NewBPlusTree() *BPlusTree {
	return NewBPlusTreeWithOrder(defaultBPlusTreeOrder)
}

func NewBTreePlus() *BPlusTree {
	return NewBPlusTree()
}

func NewBPlusTreeWithOrder(order int) *BPlusTree {
	if order < 3 {
		order = defaultBPlusTreeOrder
	}
	root := &BPlusTreeNode{isLeaf: true}
	return &BPlusTree{
		root:     root,
		leafHead: root,
		members:  make(map[string]float64),
		order:    order,
	}
}

func (b *BPlusTree) add(member string, score float64) bool {
	if oldScore, ok := b.members[member]; ok {
		if oldScore != score {
			b.update(member, score)
		}
		return false
	}

	item := bPlusItem{score: score, member: member}
	promoted, newChild := b.insertInternal(b.root, item)
	if promoted != nil {
		newRoot := &BPlusTreeNode{
			isLeaf:   false,
			keys:     []bPlusItem{*promoted},
			children: []*BPlusTreeNode{b.root, newChild},
		}
		b.root = newRoot
	}
	b.members[member] = score
	b.length++
	return true
}

func (b *BPlusTree) insertInternal(node *BPlusTreeNode, item bPlusItem) (*bPlusItem, *BPlusTreeNode) {
	if node.isLeaf {
		idx := sort.Search(len(node.items), func(i int) bool {
			return !bPlusLess(node.items[i].score, node.items[i].member, item.score, item.member)
		})
		node.items = append(node.items, bPlusItem{})
		copy(node.items[idx+1:], node.items[idx:])
		node.items[idx] = item

		if len(node.items) >= b.order {
			mid := len(node.items) / 2
			right := &BPlusTreeNode{
				isLeaf: true,
				items:  make([]bPlusItem, len(node.items)-mid),
				next:   node.next,
				prev:   node,
			}
			copy(right.items, node.items[mid:])
			if node.next != nil {
				node.next.prev = right
			}
			node.next = right
			node.items = node.items[:mid]

			promoted := right.items[0]
			return &promoted, right
		}
		return nil, nil
	}

	idx := sort.Search(len(node.keys), func(i int) bool {
		return bPlusLess(item.score, item.member, node.keys[i].score, node.keys[i].member)
	})
	promoted, newChild := b.insertInternal(node.children[idx], item)
	if promoted == nil {
		return nil, nil
	}

	node.keys = append(node.keys, bPlusItem{})
	copy(node.keys[idx+1:], node.keys[idx:])
	node.keys[idx] = *promoted

	node.children = append(node.children, nil)
	copy(node.children[idx+2:], node.children[idx+1:])
	node.children[idx+1] = newChild

	if len(node.children) > b.order {
		mid := len(node.keys) / 2
		promotedUp := node.keys[mid]

		right := &BPlusTreeNode{
			isLeaf:   false,
			keys:     make([]bPlusItem, len(node.keys)-(mid+1)),
			children: make([]*BPlusTreeNode, len(node.children)-(mid+1)),
		}
		copy(right.keys, node.keys[mid+1:])
		copy(right.children, node.children[mid+1:])

		node.keys = node.keys[:mid]
		node.children = node.children[:mid+1]

		return &promotedUp, right
	}
	return nil, nil
}

func (b *BPlusTree) update(member string, score float64) bool {
	oldScore, ok := b.members[member]
	if !ok {
		return false
	}
	if oldScore == score {
		return true
	}

	b.remove(member)
	b.add(member, score)
	return true
}

func (b *BPlusTree) remove(member string) bool {
	score, ok := b.members[member]
	if !ok {
		return false
	}

	item := bPlusItem{score: score, member: member}
	b.deleteItem(b.root, item)
	delete(b.members, member)
	b.length--

	if b.length == 0 {
		root := &BPlusTreeNode{isLeaf: true}
		b.root = root
		b.leafHead = root
	} else {
		for !b.root.isLeaf && len(b.root.children) == 1 {
			b.root = b.root.children[0]
		}
	}
	return true
}

func (b *BPlusTree) deleteItem(node *BPlusTreeNode, item bPlusItem) bool {
	if node.isLeaf {
		idx := sort.Search(len(node.items), func(i int) bool {
			return !bPlusLess(node.items[i].score, node.items[i].member, item.score, item.member)
		})
		if idx < len(node.items) && node.items[idx].member == item.member && node.items[idx].score == item.score {
			node.items = append(node.items[:idx], node.items[idx+1:]...)
			return true
		}
		return false
	}

	idx := sort.Search(len(node.keys), func(i int) bool {
		return bPlusLess(item.score, item.member, node.keys[i].score, node.keys[i].member)
	})
	removed := b.deleteItem(node.children[idx], item)
	if !removed {
		return false
	}

	if node.children[idx].isLeaf && len(node.children[idx].items) == 0 {
		emptyLeaf := node.children[idx]
		if emptyLeaf.prev != nil {
			emptyLeaf.prev.next = emptyLeaf.next
		} else {
			b.leafHead = emptyLeaf.next
		}
		if emptyLeaf.next != nil {
			emptyLeaf.next.prev = emptyLeaf.prev
		}

		node.children = append(node.children[:idx], node.children[idx+1:]...)
		if idx < len(node.keys) {
			node.keys = append(node.keys[:idx], node.keys[idx+1:]...)
		} else if len(node.keys) > 0 {
			node.keys = node.keys[:len(node.keys)-1]
		}
	} else if !node.children[idx].isLeaf && len(node.children[idx].children) == 0 {
		node.children = append(node.children[:idx], node.children[idx+1:]...)
		if idx < len(node.keys) {
			node.keys = append(node.keys[:idx], node.keys[idx+1:]...)
		} else if len(node.keys) > 0 {
			node.keys = node.keys[:len(node.keys)-1]
		}
	}
	return true
}

func (b *BPlusTree) getMemberScore(member string) (float64, bool) {
	score, ok := b.members[member]
	return score, ok
}

func (b *BPlusTree) getRank(member string) (int, bool) {
	if _, ok := b.members[member]; !ok {
		return 0, false
	}
	rank := 0
	for node := b.leafHead; node != nil; node = node.next {
		for _, item := range node.items {
			if item.member == member {
				return rank, true
			}
			rank++
		}
	}
	return 0, false
}

func (b *BPlusTree) getRange(start, stop int) []string {
	if start < 0 {
		start += b.length
	}
	if stop < 0 {
		stop += b.length
	}
	if start < 0 {
		start = 0
	}
	if stop >= b.length {
		stop = b.length - 1
	}
	if start > stop || start >= b.length {
		return []string{}
	}

	members := make([]string, 0, stop-start+1)
	index := 0
	for node := b.leafHead; node != nil && index <= stop; node = node.next {
		for _, item := range node.items {
			if index >= start && index <= stop {
				members = append(members, item.member)
			}
			index++
			if index > stop {
				break
			}
		}
	}
	return members
}

func (b *BPlusTree) isMember(member string) bool {
	_, ok := b.members[member]
	return ok
}

func (b *BPlusTree) len() int {
	return b.length
}

func (b *BPlusTree) Len() int {
	return b.length
}

func (b *BPlusTree) Add(member string, score float64) bool {
	return b.add(member, score)
}

func (b *BPlusTree) Update(member string, score float64) bool {
	return b.update(member, score)
}

func (b *BPlusTree) Remove(member string) bool {
	return b.remove(member)
}

func (b *BPlusTree) GetMemberScore(member string) (float64, bool) {
	return b.getMemberScore(member)
}

func (b *BPlusTree) GetMember(member string) (float64, bool) {
	return b.getMemberScore(member)
}

func (b *BPlusTree) GetRank(member string) (int, bool) {
	return b.getRank(member)
}

func (b *BPlusTree) GetRange(start, stop int) []string {
	return b.getRange(start, stop)
}

func (b *BPlusTree) IsMember(member string) bool {
	return b.isMember(member)
}
