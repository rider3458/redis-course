package storage

import (
	"testing"
	"time"
)

func TestStatsCountsKeys(t *testing.T) {
	s := New()
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)

	if got := s.Stats().Keys; got != 2 {
		t.Fatalf("unexpected key count: got=%d want=%d", got, 2)
	}

	s.Delete("a")
	if got := s.Stats().Keys; got != 1 {
		t.Fatalf("unexpected key count after delete: got=%d want=%d", got, 1)
	}
}

func TestStatsHitAndMiss(t *testing.T) {
	s := New()
	s.Set("k", "v", 0)

	if _, ok := s.Get("k"); !ok {
		t.Fatal("expected hit")
	}
	if _, ok := s.Get("missing"); ok {
		t.Fatal("expected miss")
	}

	stats := s.Stats()
	if stats.Hits != 1 {
		t.Fatalf("unexpected hits: got=%d want=%d", stats.Hits, 1)
	}
	if stats.Misses != 1 {
		t.Fatalf("unexpected misses: got=%d want=%d", stats.Misses, 1)
	}
}

func TestStatsUsedMemory(t *testing.T) {
	s := New()
	s.Set("key", "value", 0)

	if got, want := s.Stats().UsedMemory, int64(len("key")+len("value")); got != want {
		t.Fatalf("unexpected used memory: got=%d want=%d", got, want)
	}

	s.Set("key", "longer", 0)
	if got, want := s.Stats().UsedMemory, int64(len("key")+len("longer")); got != want {
		t.Fatalf("unexpected used memory after overwrite: got=%d want=%d", got, want)
	}

	s.Delete("key")
	if got := s.Stats().UsedMemory; got != 0 {
		t.Fatalf("unexpected used memory after delete: got=%d want=%d", got, 0)
	}
}

func TestStatsExpiredKeys(t *testing.T) {
	s := New()
	s.Set("k", "v", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	if _, ok := s.Get("k"); ok {
		t.Fatal("expected key to expire")
	}

	stats := s.Stats()
	if stats.ExpiredKeys != 1 {
		t.Fatalf("unexpected expired keys: got=%d want=%d", stats.ExpiredKeys, 1)
	}
	if stats.Misses != 1 {
		t.Fatalf("unexpected misses: got=%d want=%d", stats.Misses, 1)
	}
	if stats.UsedMemory != 0 {
		t.Fatalf("unexpected used memory: got=%d want=%d", stats.UsedMemory, 0)
	}
}

func TestStatsCountsComplexKeys(t *testing.T) {
	s := New()
	if _, err := s.SAdd("s", "ab"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}
	setWant := int64(len("s") + len("ab"))
	if got := s.Stats().UsedMemory; got != setWant {
		t.Fatalf("unexpected set memory: got=%d want=%d", got, setWant)
	}

	if _, err := s.ZAdd("z", map[string]float64{"m": 1}); err != nil {
		t.Fatalf("unexpected ZAdd error: %v", err)
	}
	zsetWant := setWant + int64(len("z")) + sortedSetMemberSize("m")
	if got := s.Stats().UsedMemory; got != zsetWant {
		t.Fatalf("unexpected zset memory: got=%d want=%d", got, zsetWant)
	}

	if err := s.CMSInit("c", 2, 2); err != nil {
		t.Fatalf("unexpected CMSInit error: %v", err)
	}
	cmsWant := zsetWant + int64(len("c")) + 2*2*8
	if got := s.Stats().UsedMemory; got != cmsWant {
		t.Fatalf("unexpected cms memory: got=%d want=%d", got, cmsWant)
	}

	if err := s.BFReserve("b", 0.01, 100); err != nil {
		t.Fatalf("unexpected BFReserve error: %v", err)
	}
	info, err := s.BFInfo("b")
	if err != nil {
		t.Fatalf("unexpected BFInfo error: %v", err)
	}
	bfWant := cmsWant + int64(len("b")) + int64(info.Size)
	if got := s.Stats().UsedMemory; got != bfWant {
		t.Fatalf("unexpected bf memory: got=%d want=%d", got, bfWant)
	}

	if got := s.Stats().Keys; got != 4 {
		t.Fatalf("unexpected key count: got=%d want=%d", got, 4)
	}
}

func TestStatsTracksComplexMutations(t *testing.T) {
	s := New()
	if _, err := s.SAdd("s", "a"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}
	base := s.Stats().UsedMemory

	if _, err := s.SAdd("s", "bb"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}
	if got, want := s.Stats().UsedMemory, base+int64(len("bb")); got != want {
		t.Fatalf("unexpected memory after SAdd: got=%d want=%d", got, want)
	}

	if _, err := s.SRem("s", "a"); err != nil {
		t.Fatalf("unexpected SRem error: %v", err)
	}
	if got, want := s.Stats().UsedMemory, base+int64(len("bb"))-int64(len("a")); got != want {
		t.Fatalf("unexpected memory after SRem: got=%d want=%d", got, want)
	}
}

func TestStatsKeysDropWhenContainerEmpties(t *testing.T) {
	s := New()
	if _, err := s.SAdd("set", "a"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}
	removed, err := s.SRem("set", "a")
	if err != nil {
		t.Fatalf("unexpected SRem error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("unexpected removed count: got=%d want=%d", removed, 1)
	}
	if got := s.Stats().Keys; got != 0 {
		t.Fatalf("unexpected key count: got=%d want=%d", got, 0)
	}
	if got := s.Stats().UsedMemory; got != 0 {
		t.Fatalf("unexpected used memory: got=%d want=%d", got, 0)
	}
}
