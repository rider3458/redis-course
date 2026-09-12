package storage

import "testing"

func TestCountMinSketchOperations(t *testing.T) {
	store := New()
	if err := store.CMSInit("visits", 100, 5); err != nil {
		t.Fatalf("CMSInit error = %v", err)
	}

	counts, err := store.CMSIncrBy("visits", []CMSIncrement{{Item: "home", Value: 3}, {Item: "about", Value: 2}, {Item: "home", Value: 4}})
	if err != nil {
		t.Fatalf("CMSIncrBy error = %v", err)
	}
	if got, want := counts, []uint64{3, 2, 7}; !equalCounts(got, want) {
		t.Fatalf("CMSIncrBy counts = %v, want %v", got, want)
	}

	counts, err = store.CMSQuery("visits", []string{"home", "about", "missing"})
	if err != nil {
		t.Fatalf("CMSQuery error = %v", err)
	}
	if got, want := counts, []uint64{7, 2, 0}; !equalCounts(got, want) {
		t.Fatalf("CMSQuery counts = %v, want %v", got, want)
	}

	info, err := store.CMSInfo("visits")
	if err != nil {
		t.Fatalf("CMSInfo error = %v", err)
	}
	if info != (CMSInfo{Width: 100, Depth: 5, Count: 9}) {
		t.Fatalf("CMSInfo = %+v", info)
	}
}

func TestCountMinSketchMerge(t *testing.T) {
	store := New()
	for _, key := range []string{"first", "second"} {
		if err := store.CMSInit(key, 100, 5); err != nil {
			t.Fatalf("CMSInit(%q) error = %v", key, err)
		}
	}
	_, _ = store.CMSIncrBy("first", []CMSIncrement{{Item: "home", Value: 3}})
	_, _ = store.CMSIncrBy("second", []CMSIncrement{{Item: "home", Value: 2}, {Item: "about", Value: 4}})

	if err := store.CMSMerge("combined", []string{"first", "second"}, []uint64{2, 1}); err != nil {
		t.Fatalf("CMSMerge error = %v", err)
	}
	counts, err := store.CMSQuery("combined", []string{"home", "about"})
	if err != nil {
		t.Fatalf("CMSQuery error = %v", err)
	}
	if got, want := counts, []uint64{8, 4}; !equalCounts(got, want) {
		t.Fatalf("CMSQuery counts = %v, want %v", got, want)
	}
}

func TestCountMinSketchRejectsInvalidKeysAndTypes(t *testing.T) {
	store := New()
	if err := store.CMSInit("sketch", 0, 1); err == nil {
		t.Fatal("CMSInit accepted zero width")
	}
	if err := store.CMSInit("sketch", 10, 2); err != nil {
		t.Fatalf("CMSInit error = %v", err)
	}
	if err := store.CMSInit("sketch", 10, 2); err != ErrKeyExists {
		t.Fatalf("second CMSInit error = %v, want %v", err, ErrKeyExists)
	}
	if _, err := store.CMSQuery("missing", []string{"item"}); err != ErrCMSKeyNotFound {
		t.Fatalf("CMSQuery missing error = %v", err)
	}
	store.Set("string", "value", 0)
	if _, err := store.CMSQuery("string", []string{"item"}); err != ErrWrongType {
		t.Fatalf("CMSQuery wrong type error = %v", err)
	}
}

func equalCounts(got, want []uint64) bool {
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
