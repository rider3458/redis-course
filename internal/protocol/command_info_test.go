package protocol

import (
	"strings"
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleINFOReturnsDefaultSections(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	got := string(HandleINFO(&Command{Cmd: "INFO"}))
	if !strings.HasPrefix(got, "$") {
		t.Fatalf("expected bulk string response, got %q", got)
	}
	for _, want := range []string{
		"# Memory",
		"# Stats",
		"# Keyspace",
		"used_memory:0",
		"keyspace_hits:0",
		"maxmemory_policy:lru",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in INFO output, got %q", want, got)
		}
	}
}

func TestHandleINFOFiltersRequestedSection(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	got := string(HandleINFO(&Command{Args: []string{"STATS"}}))
	if !strings.Contains(got, "keyspace_hits:0") {
		t.Fatalf("expected stats section, got %q", got)
	}
	if strings.Contains(got, "# Memory") || strings.Contains(got, "# Keyspace") {
		t.Fatalf("expected only the stats section, got %q", got)
	}
}

func TestHandleINFOUnknownSectionIsEmpty(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	if got, want := string(HandleINFO(&Command{Args: []string{"bogus"}})), "$0\r\n\r\n"; got != want {
		t.Fatalf("INFO bogus = %q, want %q", got, want)
	}
}

func TestHandleINFOReportsStoreStats(t *testing.T) {
	store := storage.New(storage.WithMaxMemory(1024), storage.WithLFU())
	restore := replaceCommandStoreForTest(store)
	t.Cleanup(restore)

	store.Set("k", "v", 0)
	store.Get("k")
	store.Get("missing")

	got := string(HandleINFO(&Command{Args: []string{"memory", "stats"}}))
	for _, want := range []string{
		"used_memory:2",
		"maxmemory:1024",
		"maxmemory_policy:lfu",
		"keyspace_hits:1",
		"keyspace_misses:1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in INFO output, got %q", want, got)
		}
	}

	keyspace := string(HandleINFO(&Command{Args: []string{"keyspace"}}))
	if !strings.Contains(keyspace, "db0:keys=1,expires=0,avg_ttl=0") {
		t.Fatalf("expected keyspace detail, got %q", keyspace)
	}
}

func TestHandleINFORoutesThroughDispatch(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	got := string(DispatchCommand(&Command{Cmd: "INFO", Args: []string{"stats"}}))
	if !strings.Contains(got, "# Stats") {
		t.Fatalf("expected INFO dispatched to handler, got %q", got)
	}
}
