package storage

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

func TestBPlusTreeBasicOperations(t *testing.T) {
	tree := NewBPlusTree()

	if tree.len() != 0 {
		t.Fatalf("expected length 0, got %d", tree.len())
	}

	// Add new elements
	if !tree.add("alice", 20) {
		t.Fatal("expected alice to be added")
	}
	if !tree.add("bob", 10.5) {
		t.Fatal("expected bob to be added")
	}
	if !tree.add("carol", 20) {
		t.Fatal("expected carol to be added")
	}

	if tree.len() != 3 {
		t.Fatalf("expected length 3, got %d", tree.len())
	}

	// Adding existing member with same score should return false
	if tree.add("alice", 20) {
		t.Fatal("expected adding duplicate alice with same score to return false")
	}

	// Check scores
	score, ok := tree.getMemberScore("bob")
	if !ok || score != 10.5 {
		t.Fatalf("unexpected bob score: got %g, ok %t", score, ok)
	}
	score, ok = tree.getMemberScore("alice")
	if !ok || score != 20 {
		t.Fatalf("unexpected alice score: got %g, ok %t", score, ok)
	}
	_, ok = tree.getMemberScore("nonexistent")
	if ok {
		t.Fatal("expected nonexistent member to not be found")
	}

	// Check membership
	if !tree.isMember("alice") || !tree.isMember("bob") || !tree.isMember("carol") {
		t.Fatal("expected all members to be present")
	}
	if tree.isMember("david") {
		t.Fatal("expected david to not be present")
	}

	// Check ranks (ordered: bob: 10.5, alice: 20, carol: 20)
	rank, ok := tree.getRank("bob")
	if !ok || rank != 0 {
		t.Fatalf("unexpected bob rank: got %d, ok %t", rank, ok)
	}
	rank, ok = tree.getRank("alice")
	if !ok || rank != 1 {
		t.Fatalf("unexpected alice rank: got %d, ok %t", rank, ok)
	}
	rank, ok = tree.getRank("carol")
	if !ok || rank != 2 {
		t.Fatalf("unexpected carol rank: got %d, ok %t", rank, ok)
	}

	// Update alice score to 5 (becomes rank 0)
	if tree.add("alice", 5) {
		t.Fatal("expected update to return false from add")
	}
	rank, ok = tree.getRank("alice")
	if !ok || rank != 0 {
		t.Fatalf("unexpected alice rank after update: got %d, ok %t", rank, ok)
	}
	rank, ok = tree.getRank("bob")
	if !ok || rank != 1 {
		t.Fatalf("unexpected bob rank after alice update: got %d, ok %t", rank, ok)
	}

	// Range check
	rangeMembers := tree.getRange(0, -1)
	expected := []string{"alice", "bob", "carol"}
	if len(rangeMembers) != len(expected) {
		t.Fatalf("unexpected range length: got %d, want %d", len(rangeMembers), len(expected))
	}
	for i, name := range expected {
		if rangeMembers[i] != name {
			t.Fatalf("unexpected range[%d]: got %s, want %s", i, rangeMembers[i], name)
		}
	}

	// Remove member
	if !tree.remove("bob") {
		t.Fatal("expected bob removal to return true")
	}
	if tree.remove("bob") {
		t.Fatal("expected second bob removal to return false")
	}
	if tree.len() != 2 {
		t.Fatalf("expected length 2, got %d", tree.len())
	}
	if tree.isMember("bob") {
		t.Fatal("expected bob to be removed")
	}
}

func TestBPlusTreeSmallOrderSplitsAndRange(t *testing.T) {
	// Use order 3 to force frequent splits and deep tree
	tree := NewBPlusTreeWithOrder(3)

	n := 100
	for i := 0; i < n; i++ {
		member := fmt.Sprintf("m%03d", i)
		score := float64(i * 10)
		if !tree.add(member, score) {
			t.Fatalf("failed to add %s", member)
		}
	}

	if tree.len() != n {
		t.Fatalf("expected length %d, got %d", n, tree.len())
	}

	// Verify ranks and scores
	for i := 0; i < n; i++ {
		member := fmt.Sprintf("m%03d", i)
		score, ok := tree.getMemberScore(member)
		if !ok || score != float64(i*10) {
			t.Fatalf("unexpected score for %s: got %g, ok %t", member, score, ok)
		}
		rank, ok := tree.getRank(member)
		if !ok || rank != i {
			t.Fatalf("unexpected rank for %s: got %d, want %d, ok %t", member, rank, i, ok)
		}
	}

	// Check range
	got := tree.getRange(10, 20)
	if len(got) != 11 {
		t.Fatalf("unexpected range length: got %d, want 11", len(got))
	}
	for i, m := range got {
		expected := fmt.Sprintf("m%03d", 10+i)
		if m != expected {
			t.Fatalf("range element %d mismatch: got %s, want %s", i, m, expected)
		}
	}

	// Remove half elements
	for i := 0; i < n; i += 2 {
		member := fmt.Sprintf("m%03d", i)
		if !tree.remove(member) {
			t.Fatalf("failed to remove %s", member)
		}
	}

	if tree.len() != n/2 {
		t.Fatalf("expected length %d, got %d", n/2, tree.len())
	}

	// Verify remaining elements
	remaining := tree.getRange(0, -1)
	if len(remaining) != n/2 {
		t.Fatalf("unexpected remaining range length: got %d, want %d", len(remaining), n/2)
	}
	for idx, m := range remaining {
		expected := fmt.Sprintf("m%03d", idx*2+1)
		if m != expected {
			t.Fatalf("remaining element %d mismatch: got %s, want %s", idx, m, expected)
		}
	}

	// Remove all remaining
	for i := 1; i < n; i += 2 {
		member := fmt.Sprintf("m%03d", i)
		if !tree.remove(member) {
			t.Fatalf("failed to remove %s", member)
		}
	}

	if tree.len() != 0 {
		t.Fatalf("expected length 0 after removing all, got %d", tree.len())
	}
	if len(tree.getRange(0, -1)) != 0 {
		t.Fatal("expected empty range for empty tree")
	}
}

func TestBPlusTreeRandomOrder(t *testing.T) {
	tree := NewBPlusTreeWithOrder(4)

	type entry struct {
		member string
		score  float64
	}

	entries := make([]entry, 50)
	for i := range entries {
		entries[i] = entry{
			member: fmt.Sprintf("item_%02d", i),
			score:  float64(i),
		}
	}

	// Shuffle
	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})

	for _, e := range entries {
		tree.add(e.member, e.score)
	}

	if tree.len() != len(entries) {
		t.Fatalf("expected length %d, got %d", len(entries), tree.len())
	}

	// In-order traversal via getRange should give items sorted by score
	all := tree.getRange(0, -1)
	for i := 0; i < len(all); i++ {
		expected := fmt.Sprintf("item_%02d", i)
		if all[i] != expected {
			t.Fatalf("item at index %d: got %s, want %s", i, all[i], expected)
		}
	}
}
