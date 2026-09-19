package storage

import (
	"testing"
	"time"
)

func TestEvictWithoutLimitsIsNoop(t *testing.T) {
	s := New()
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)

	if got := s.Evict(); got != 0 {
		t.Fatalf("unexpected evicted count: got=%d want=%d", got, 0)
	}
	if got := s.Stats().Keys; got != 2 {
		t.Fatalf("unexpected key count: got=%d want=%d", got, 2)
	}
}

func TestDefaultPolicyEvictsLRU(t *testing.T) {
	s := New(WithMaxEntries(2))
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)
	s.Set("c", "3", 0)

	if got := s.Stats().Keys; got != 2 {
		t.Fatalf("unexpected key count: got=%d want=%d", got, 2)
	}
	if s.Exists("a") != 0 {
		t.Fatal("expected default LRU policy to evict key a")
	}
	if s.Exists("b") != 1 || s.Exists("c") != 1 {
		t.Fatal("expected keys b and c to remain")
	}
	if got := s.Stats().EvictedKeys; got != 1 {
		t.Fatalf("unexpected evicted count: got=%d want=%d", got, 1)
	}
}

func TestMaxEntriesEvictsLRU(t *testing.T) {
	s := New(WithMaxEntries(2), WithLRU())
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)
	// Touch a so b becomes the least recently used key.
	s.Get("a")
	s.Set("c", "3", 0)

	if s.Exists("b") != 0 {
		t.Fatal("expected least recently used key b to be evicted")
	}
	if s.Exists("a") != 1 || s.Exists("c") != 1 {
		t.Fatal("expected keys a and c to remain")
	}
}

func TestMaxEntriesEvictsLFU(t *testing.T) {
	s := New(WithMaxEntries(2), WithLFU())
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)
	// Raise a's frequency so b is the least frequently used key.
	s.Get("a")
	s.Get("a")
	s.Set("c", "3", 0)

	if s.Exists("b") != 0 {
		t.Fatal("expected least frequently used key b to be evicted")
	}
	if s.Exists("a") != 1 || s.Exists("c") != 1 {
		t.Fatal("expected keys a and c to remain")
	}
}

func TestMaxEntriesEvictsWithRandomPolicy(t *testing.T) {
	s := New(WithMaxEntries(1), WithRandomEviction())
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)
	s.Set("c", "3", 0)

	if got := s.Stats().Keys; got != 1 {
		t.Fatalf("unexpected key count: got=%d want=%d", got, 1)
	}
	if got := s.Stats().EvictedKeys; got != 2 {
		t.Fatalf("unexpected evicted count: got=%d want=%d", got, 2)
	}
}

func TestMaxMemoryEvictsKeys(t *testing.T) {
	s := New(WithMaxMemory(4), WithLRU())
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)
	s.Set("c", "3", 0)

	if got := s.Stats().UsedMemory; got > 4 {
		t.Fatalf("used memory exceeds limit: got=%d want<=%d", got, 4)
	}
	if s.Exists("a") != 0 {
		t.Fatal("expected key a to be evicted")
	}
}

func TestMaxMemoryEvictsComplexKeys(t *testing.T) {
	s := New(WithMaxMemory(10), WithLRU())
	if err := s.CMSInit("cms", 2, 2); err != nil {
		t.Fatalf("unexpected CMSInit error: %v", err)
	}

	info, err := s.CMSInfo("cms")
	if err == nil {
		t.Fatalf("expected cms key to be evicted, got info %#v", info)
	}
	if got := s.Stats().EvictedKeys; got != 1 {
		t.Fatalf("unexpected evicted count: got=%d want=%d", got, 1)
	}
	if got := s.Stats().UsedMemory; got > 10 {
		t.Fatalf("used memory exceeds limit: got=%d want<=%d", got, 10)
	}
}

func TestMaxMemoryEvictsWhenSetMembersGrow(t *testing.T) {
	s := New(WithMaxMemory(12), WithLRU())
	if _, err := s.SAdd("s", "aaa"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}
	if _, err := s.SAdd("s", "bbbbbbbbbb"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}

	if got := s.Stats().UsedMemory; got > 12 {
		t.Fatalf("used memory exceeds limit: got=%d want<=%d", got, 12)
	}
	if got := s.Stats().Keys; got != 0 {
		t.Fatalf("expected the oversized set to be evicted, got %d keys", got)
	}
}

func TestMaxMemoryEvictsWhenSortedSetMembersGrow(t *testing.T) {
	s := New(WithMaxMemory(10), WithLRU())
	if _, err := s.ZAdd("z", map[string]float64{"m": 1}); err != nil {
		t.Fatalf("unexpected ZAdd error: %v", err)
	}
	if _, err := s.ZAdd("z", map[string]float64{"mm": 2}); err != nil {
		t.Fatalf("unexpected ZAdd error: %v", err)
	}

	if got := s.Stats().UsedMemory; got > 10 {
		t.Fatalf("used memory exceeds limit: got=%d want<=%d", got, 10)
	}
	if got := s.Stats().Keys; got != 0 {
		t.Fatalf("expected the oversized sorted set to be evicted, got %d keys", got)
	}
}

func TestMaxMemoryEvictsWhenMergeOverwriteGrows(t *testing.T) {
	s := New(WithMaxMemory(50), WithLRU())
	if err := s.CMSInit("a", 2, 2); err != nil {
		t.Fatalf("unexpected CMSInit error: %v", err)
	}
	if err := s.CMSInit("c", 1, 1); err != nil {
		t.Fatalf("unexpected CMSInit error: %v", err)
	}
	if err := s.CMSMerge("c", []string{"a"}, []uint64{1}); err != nil {
		t.Fatalf("unexpected CMSMerge error: %v", err)
	}

	stats := s.Stats()
	if stats.UsedMemory > 50 {
		t.Fatalf("used memory exceeds limit: got=%d want<=%d", stats.UsedMemory, 50)
	}
	if stats.EvictedKeys != 1 {
		t.Fatalf("unexpected evicted count: got=%d want=%d", stats.EvictedKeys, 1)
	}
	if stats.Keys != 1 {
		t.Fatalf("unexpected key count: got=%d want=%d", stats.Keys, 1)
	}
}

func TestEvictionPolicyNames(t *testing.T) {
	tests := []struct {
		name   string
		policy EvictionPolicy
		want   string
	}{
		{name: "lru", policy: NewLRU(), want: "lru"},
		{name: "random", policy: NewRandomEviction(), want: "random"},
		{name: "lfu", policy: NewLFU(), want: "lfu"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.policy.Name(); got != tt.want {
				t.Fatalf("unexpected policy name: got=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestLRUTracksExactOrder(t *testing.T) {
	policy := NewLRU()
	for _, key := range []string{"k1", "k2", "k3", "k4", "k5", "k6"} {
		policy.OnInsert(key)
	}
	// Touch k1..k5 so k6 is the least recently used key.
	for _, key := range []string{"k1", "k2", "k3", "k4", "k5"} {
		policy.OnAccess(key)
	}

	if got, ok := policy.Victim(); !ok || got != "k6" {
		t.Fatalf("unexpected victim: got=%q ok=%v want=%q", got, ok, "k6")
	}

	// After removing k6, k1 is the least recently used remaining key.
	policy.OnRemove("k6")
	if got, ok := policy.Victim(); !ok || got != "k1" {
		t.Fatalf("unexpected victim after remove: got=%q ok=%v want=%q", got, ok, "k1")
	}
}

func TestEvictionPolicyByName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantNil  bool
		wantOK   bool
	}{
		{name: "lru", input: "lru", wantName: "lru", wantOK: true},
		{name: "allkeys lru", input: "allkeys-lru", wantName: "lru", wantOK: true},
		{name: "random is case insensitive", input: " RANDOM ", wantName: "random", wantOK: true},
		{name: "lfu", input: "lfu", wantName: "lfu", wantOK: true},
		{name: "noeviction", input: "noeviction", wantNil: true, wantOK: true},
		{name: "unknown", input: "bogus", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, ok := EvictionPolicyByName(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("unexpected ok: got=%v want=%v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if tt.wantNil {
				if policy != nil {
					t.Fatalf("expected nil policy, got %q", policy.Name())
				}
				return
			}
			if policy == nil || policy.Name() != tt.wantName {
				t.Fatalf("unexpected policy: got=%v want=%q", policy, tt.wantName)
			}
		})
	}
}

type recordingPolicy struct {
	inserts  []string
	accesses []string
	removes  []string
}

func (p *recordingPolicy) Name() string { return "recording" }

func (p *recordingPolicy) OnInsert(key string) { p.inserts = append(p.inserts, key) }
func (p *recordingPolicy) OnAccess(key string) { p.accesses = append(p.accesses, key) }
func (p *recordingPolicy) OnRemove(key string) { p.removes = append(p.removes, key) }

func (p *recordingPolicy) Victim() (string, bool) { return "", false }

func TestStoreNotifiesEvictionPolicy(t *testing.T) {
	policy := &recordingPolicy{}
	s := New(WithEvictionPolicy(policy))

	s.Set("a", "1", 0)
	s.Get("a")
	s.Set("a", "2", 0)
	s.Incr("n")
	s.Incr("n")
	s.Delete("a")

	if len(policy.inserts) != 2 || policy.inserts[0] != "a" || policy.inserts[1] != "n" {
		t.Fatalf("unexpected inserts: %#v", policy.inserts)
	}
	if len(policy.accesses) != 3 {
		t.Fatalf("unexpected accesses: %#v", policy.accesses)
	}
	if len(policy.removes) != 1 || policy.removes[0] != "a" {
		t.Fatalf("unexpected removes: %#v", policy.removes)
	}
}

func TestStoreNotifiesPolicyOnExpiry(t *testing.T) {
	policy := &recordingPolicy{}
	s := New(WithEvictionPolicy(policy))

	s.Set("k", "v", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	if _, ok := s.Get("k"); ok {
		t.Fatal("expected key to expire")
	}
	if len(policy.removes) != 1 || policy.removes[0] != "k" {
		t.Fatalf("unexpected removes: %#v", policy.removes)
	}
}

func TestMaxEntriesEvictsComplexKeys(t *testing.T) {
	s := New(WithMaxEntries(1), WithLRU())
	if _, err := s.SAdd("set", "a"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}
	if _, err := s.ZAdd("zset", map[string]float64{"m": 1}); err != nil {
		t.Fatalf("unexpected ZAdd error: %v", err)
	}

	if got := s.Stats().Keys; got != 1 {
		t.Fatalf("unexpected key count: got=%d want=%d", got, 1)
	}
	if got := s.Stats().EvictedKeys; got != 1 {
		t.Fatalf("unexpected evicted count: got=%d want=%d", got, 1)
	}
	if s.Exists("set") != 0 {
		t.Fatal("expected least recently used set key to be evicted")
	}
}

func TestStoreNotifiesPolicyOnComplexAccess(t *testing.T) {
	policy := &recordingPolicy{}
	s := New(WithEvictionPolicy(policy))

	if _, err := s.SAdd("set", "a"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}
	if _, err := s.ZAdd("zset", map[string]float64{"m": 1}); err != nil {
		t.Fatalf("unexpected ZAdd error: %v", err)
	}
	if err := s.CMSInit("cms", 4, 2); err != nil {
		t.Fatalf("unexpected CMSInit error: %v", err)
	}
	if err := s.BFReserve("bf", 0.01, 100); err != nil {
		t.Fatalf("unexpected BFReserve error: %v", err)
	}

	if _, err := s.SIsMember("set", "a"); err != nil {
		t.Fatalf("unexpected SIsMember error: %v", err)
	}
	if _, _, err := s.ZScore("zset", "m"); err != nil {
		t.Fatalf("unexpected ZScore error: %v", err)
	}
	if _, err := s.CMSQuery("cms", []string{"x"}); err != nil {
		t.Fatalf("unexpected CMSQuery error: %v", err)
	}
	if _, err := s.BFExists("bf", "x"); err != nil {
		t.Fatalf("unexpected BFExists error: %v", err)
	}

	if len(policy.inserts) != 4 {
		t.Fatalf("unexpected inserts: %#v", policy.inserts)
	}
	if len(policy.accesses) != 4 {
		t.Fatalf("unexpected accesses: %#v", policy.accesses)
	}
}
