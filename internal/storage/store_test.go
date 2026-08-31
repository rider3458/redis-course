package storage

import (
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	s := New()
	s.Set("k", "v", 0)

	got, ok := s.Get("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != "v" {
		t.Fatalf("unexpected value: got=%q want=%q", got, "v")
	}
}

func TestTTLExpiry(t *testing.T) {
	s := New()
	s.Set("k", "v", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	_, ok := s.Get("k")
	if ok {
		t.Fatal("expected key to expire")
	}
}

func TestDelete(t *testing.T) {
	s := New()
	s.Set("a", "1", 0)
	s.Set("b", "2", 0)

	deleted := s.Delete("a", "c")
	if deleted != 1 {
		t.Fatalf("unexpected deleted count: got=%d want=%d", deleted, 1)
	}

	if s.Exists("a") != 0 {
		t.Fatal("expected key a to be deleted")
	}
	if s.Exists("b") != 1 {
		t.Fatal("expected key b to exist")
	}
}

func TestExpire(t *testing.T) {
	s := New()
	s.Set("k", "v", 0)

	ok := s.Expire("k", 10*time.Millisecond)
	if !ok {
		t.Fatal("expected expire to succeed for existing key")
	}
	time.Sleep(20 * time.Millisecond)

	if s.Exists("k") != 0 {
		t.Fatal("expected key to expire")
	}
}

func TestExpireMissingKeyReturnsFalse(t *testing.T) {
	s := New()

	ok := s.Expire("missing", time.Second)
	if ok {
		t.Fatal("expected expire to fail for missing key")
	}
}

func TestTTLNoExpireReturnsMinusOne(t *testing.T) {
	s := New()
	s.Set("k", "v", 0)

	got := s.TTL("k")
	if got != -1 {
		t.Fatalf("unexpected TTL: got=%d want=%d", got, -1)
	}
}

func TestTTLMissingReturnsMinusTwo(t *testing.T) {
	s := New()

	got := s.TTL("missing")
	if got != -2 {
		t.Fatalf("unexpected TTL: got=%d want=%d", got, -2)
	}
}

func TestTTLForExpiringKey(t *testing.T) {
	s := New()
	s.Set("k", "v", 5*time.Second)

	got := s.TTL("k")
	if got < 0 || got > 5 {
		t.Fatalf("unexpected TTL range: got=%d", got)
	}
}

func TestTTLExpiredReturnsMinusTwo(t *testing.T) {
	s := New()
	s.Set("k", "v", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	got := s.TTL("k")
	if got != -2 {
		t.Fatalf("unexpected TTL: got=%d want=%d", got, -2)
	}
}

func TestIncr(t *testing.T) {
	s := New()

	v, err := s.Incr("n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 1 {
		t.Fatalf("unexpected value: got=%d want=%d", v, 1)
	}

	s.Set("n", "10", 0)
	v, err = s.Incr("n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 11 {
		t.Fatalf("unexpected value: got=%d want=%d", v, 11)
	}
}

func TestIncrRejectsNonInteger(t *testing.T) {
	s := New()
	s.Set("n", "hello", 0)

	_, err := s.Incr("n")
	if err == nil {
		t.Fatal("expected error for non-integer value")
	}
	if err != ErrNotInteger {
		t.Fatalf("unexpected error: got=%v want=%v", err, ErrNotInteger)
	}
}

func TestSAddAndSMembers(t *testing.T) {
	s := New()

	added, err := s.SAdd("letters", "a", "b", "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added != 2 {
		t.Fatalf("unexpected added count: got=%d want=%d", added, 2)
	}

	members, err := s.SMembers("letters")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 || members[0] != "a" || members[1] != "b" {
		t.Fatalf("unexpected members: %#v", members)
	}
}

func TestSRemAndSIsMember(t *testing.T) {
	s := New()
	_, _ = s.SAdd("letters", "a", "b", "c")

	removed, err := s.SRem("letters", "b", "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("unexpected removed count: got=%d want=%d", removed, 1)
	}

	member, err := s.SIsMember("letters", "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if member != 0 {
		t.Fatalf("unexpected membership: got=%d want=%d", member, 0)
	}
}

func TestSetOpsOnStringKeyReturnWrongType(t *testing.T) {
	s := New()
	s.Set("k", "v", 0)

	if _, err := s.SAdd("k", "a"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from SAdd, got %v", err)
	}
	if _, err := s.SRem("k", "a"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from SRem, got %v", err)
	}
	if _, err := s.SIsMember("k", "a"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from SIsMember, got %v", err)
	}
	if _, err := s.SMembers("k"); err != ErrWrongType {
		t.Fatalf("expected ErrWrongType from SMembers, got %v", err)
	}
}

func TestExistsCountsSetKeys(t *testing.T) {
	s := New()
	_, _ = s.SAdd("letters", "a")

	count := s.Exists("letters")
	if count != 1 {
		t.Fatalf("unexpected exists count: got=%d want=%d", count, 1)
	}
}
