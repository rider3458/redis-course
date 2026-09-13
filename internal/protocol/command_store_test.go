package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestConfigureCommandStore(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	ConfigureCommandStore(storage.WithMaxMemory(1024), storage.WithLFU())

	stats := commandStore().Stats()
	if stats.MaxMemory != 1024 {
		t.Fatalf("unexpected max memory: got=%d want=%d", stats.MaxMemory, 1024)
	}
	if stats.EvictionPolicy != "lfu" {
		t.Fatalf("unexpected policy: got=%q want=%q", stats.EvictionPolicy, "lfu")
	}
}

func TestConfigureCommandStoreNoEviction(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	ConfigureCommandStore(storage.WithEvictionPolicy(nil))

	if got := commandStore().Stats().EvictionPolicy; got != "noeviction" {
		t.Fatalf("unexpected policy: got=%q want=%q", got, "noeviction")
	}
}
