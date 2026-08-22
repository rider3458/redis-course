package protocol

import (
	"sync"

	"github.com/rider3458/redis-course/internal/storage"
)

var (
	storeMu     sync.RWMutex
	sharedStore = storage.New()
)

func commandStore() *storage.Store {
	storeMu.RLock()
	store := sharedStore
	storeMu.RUnlock()
	return store
}

// replaceCommandStoreForTest swaps the shared store and returns a restore function.
//
// It is intentionally unexported and intended for package-local tests only.
func replaceCommandStoreForTest(store *storage.Store) func() {
	if store == nil {
		panic("store must not be nil")
	}

	storeMu.Lock()
	previous := sharedStore
	sharedStore = store
	storeMu.Unlock()

	return func() {
		storeMu.Lock()
		sharedStore = previous
		storeMu.Unlock()
	}
}
