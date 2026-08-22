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
