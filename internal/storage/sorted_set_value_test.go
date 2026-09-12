package storage

import (
	"fmt"
	"sync"
	"testing"
)

func TestSortedSetOperations(t *testing.T) {
	for _, opt := range []Option{WithSkipList(), WithBPlusTree()} {
		store := New(opt)

		added, err := store.ZAdd("scores", map[string]float64{"alice": 20, "bob": 10.5, "carol": 20})
		if err != nil || added != 3 {
			t.Fatalf("unexpected ZADD result: added=%d err=%v", added, err)
		}

		rank, found, err := store.ZRank("scores", "bob")
		if err != nil || !found || rank != 0 {
			t.Fatalf("unexpected bob rank: rank=%d found=%t err=%v", rank, found, err)
		}
		rank, found, err = store.ZRank("scores", "carol")
		if err != nil || !found || rank != 2 {
			t.Fatalf("unexpected carol rank: rank=%d found=%t err=%v", rank, found, err)
		}

		added, err = store.ZAdd("scores", map[string]float64{"alice": 5})
		if err != nil || added != 0 {
			t.Fatalf("unexpected update ZADD result: added=%d err=%v", added, err)
		}
		score, found, err := store.ZScore("scores", "alice")
		if err != nil || !found || score != 5 {
			t.Fatalf("unexpected alice score: score=%g found=%t err=%v", score, found, err)
		}
		rank, found, err = store.ZRank("scores", "alice")
		if err != nil || !found || rank != 0 {
			t.Fatalf("unexpected alice rank after update: rank=%d found=%t err=%v", rank, found, err)
		}

		removed, err := store.ZRem("scores", "alice", "missing")
		if err != nil || removed != 1 {
			t.Fatalf("unexpected ZREM result: removed=%d err=%v", removed, err)
		}
		_, found, err = store.ZScore("scores", "alice")
		if err != nil || found {
			t.Fatalf("expected alice to be absent: found=%t err=%v", found, err)
		}
	}
}

func TestSortedSetRejectsOtherValueTypes(t *testing.T) {
	store := New()
	store.Set("key", "value", 0)

	if _, err := store.ZAdd("key", map[string]float64{"member": 1}); err != ErrWrongType {
		t.Fatalf("unexpected ZADD error: %v", err)
	}
	if _, _, err := store.ZScore("key", "member"); err != ErrWrongType {
		t.Fatalf("unexpected ZSCORE error: %v", err)
	}
	if _, _, err := store.ZRank("key", "member"); err != ErrWrongType {
		t.Fatalf("unexpected ZRANK error: %v", err)
	}
	if _, err := store.ZRem("key", "member"); err != ErrWrongType {
		t.Fatalf("unexpected ZREM error: %v", err)
	}
}

func TestSortedSetRange(t *testing.T) {
	for _, opt := range []Option{WithSkipList(), WithBPlusTree()} {
		store := New(opt)
		_, err := store.ZAdd("scores", map[string]float64{"alice": 3, "bob": 1, "carol": 2})
		if err != nil {
			t.Fatalf("unexpected ZADD error: %v", err)
		}

		assertRange(t, store, "scores", 0, -1, []string{"bob", "carol", "alice"})
		assertRange(t, store, "scores", 1, 5, []string{"carol", "alice"})
		assertRange(t, store, "scores", -2, -1, []string{"carol", "alice"})
		assertRange(t, store, "scores", 3, 1, []string{})
		assertRange(t, store, "missing", 0, -1, []string{})
	}
}

func TestSortedSetTypeOption(t *testing.T) {
	store := New()
	if store.SortedSetType() != SortedSetTypeSkipList {
		t.Fatalf("expected default type to be SkipList, got %v", store.SortedSetType())
	}

	store.SetSortedSetType(SortedSetTypeBPlusTree)
	if store.SortedSetType() != SortedSetTypeBPlusTree {
		t.Fatalf("expected type to be BPlusTree, got %v", store.SortedSetType())
	}

	set1 := NewSortedSet(SortedSetTypeSkipList)
	if _, ok := set1.Index().(*SkipList); !ok {
		t.Fatalf("expected SkipList index, got %T", set1.Index())
	}

	set2 := NewSortedSet(SortedSetTypeBPlusTree)
	if _, ok := set2.Index().(*BPlusTree); !ok {
		t.Fatalf("expected BPlusTree index, got %T", set2.Index())
	}
}

func TestSortedSetConcurrentAccess(t *testing.T) {
	for _, opt := range []Option{WithSkipList(), WithBPlusTree()} {
		store := New(opt)
		if _, err := store.ZAdd("scores", map[string]float64{"member": 0}); err != nil {
			t.Fatalf("ZAdd setup error: %v", err)
		}

		var group sync.WaitGroup
		for worker := range 4 {
			group.Add(1)
			go func(worker int) {
				defer group.Done()
				member := fmt.Sprintf("member-%d", worker)
				for score := range 100 {
					if _, err := store.ZAdd("scores", map[string]float64{member: float64(score)}); err != nil {
						t.Errorf("ZAdd error: %v", err)
					}
					if _, _, err := store.ZScore("scores", member); err != nil {
						t.Errorf("ZScore error: %v", err)
					}
					if _, _, err := store.ZRank("scores", member); err != nil {
						t.Errorf("ZRank error: %v", err)
					}
					if _, err := store.ZRange("scores", 0, -1); err != nil {
						t.Errorf("ZRange error: %v", err)
					}
				}
			}(worker)
		}
		group.Wait()
	}
}

func assertRange(t *testing.T, store *Store, key string, start, stop int, want []string) {
	t.Helper()
	got, err := store.ZRange(key, start, stop)
	if err != nil {
		t.Fatalf("unexpected ZRANGE error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected ZRANGE length: got=%v want=%v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected ZRANGE result: got=%v want=%v", got, want)
		}
	}
}
