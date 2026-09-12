package storage

import "testing"

func TestBloomFilterOperations(t *testing.T) {
	store := New()
	if err := store.BFReserve("seen", 0.01, 100); err != nil {
		t.Fatalf("BFReserve error = %v", err)
	}

	added, err := store.BFAdd("seen", "home")
	if err != nil || !added {
		t.Fatalf("first BFAdd = %t, %v; want true, nil", added, err)
	}
	added, err = store.BFAdd("seen", "home")
	if err != nil || added {
		t.Fatalf("second BFAdd = %t, %v; want false, nil", added, err)
	}

	results, err := store.BFMultiAdd("seen", []string{"about", "contact"})
	if err != nil || !equalBools(results, []bool{true, true}) {
		t.Fatalf("BFMultiAdd = %v, %v", results, err)
	}
	results, err = store.BFMultiExists("seen", []string{"home", "about", "missing"})
	if err != nil || !equalBools(results, []bool{true, true, false}) {
		t.Fatalf("BFMultiExists = %v, %v", results, err)
	}

	info, err := store.BFInfo("seen")
	if err != nil {
		t.Fatalf("BFInfo error = %v", err)
	}
	if info.Capacity != 100 || info.Count != 3 || info.Size == 0 || info.HashCount == 0 {
		t.Fatalf("BFInfo = %+v", info)
	}
}

func TestBloomFilterDefaultsAndErrors(t *testing.T) {
	store := New()
	added, err := store.BFAdd("new", "item")
	if err != nil || !added {
		t.Fatalf("BFAdd default = %t, %v; want true, nil", added, err)
	}
	if err := store.BFReserve("invalid", 0, 100); err != ErrBloomFilterIncompatible {
		t.Fatalf("BFReserve invalid error = %v", err)
	}
	if _, err := store.BFInfo("missing"); err != ErrBloomFilterKeyNotFound {
		t.Fatalf("BFInfo missing error = %v", err)
	}
	store.Set("string", "value", 0)
	if _, err := store.BFExists("string", "item"); err != ErrWrongType {
		t.Fatalf("BFExists wrong type error = %v", err)
	}
}

func equalBools(got, want []bool) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
