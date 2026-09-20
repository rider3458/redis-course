package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rider3458/redis-course/internal/protocol"
	server "github.com/rider3458/redis-course/internal/server"
	"github.com/rider3458/redis-course/internal/storage"
)

func main() {
	address := flag.String("addr", ":8080", "listen address")
	maxEntries := flag.Int("maxentries", 0, "evict when live keys exceed this count (0 = unlimited)")
	maxMemory := flag.Int64("maxmemory", 0, "evict when approximate memory exceeds this many bytes (0 = unlimited)")
	policyName := flag.String("maxmemory-policy", "lru", "eviction policy: lru, random, lfu, or noeviction")
	sweepInterval := flag.Duration("sweep-interval", 100*time.Millisecond, "how often to expire stale keys and enforce capacity (0 = disabled)")
	flag.Parse()

	policy, ok := storage.EvictionPolicyByName(*policyName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown -maxmemory-policy %q\n", *policyName)
		os.Exit(1)
	}

	store := protocol.ConfigureCommandStore(
		storage.WithMaxEntries(*maxEntries),
		storage.WithMaxMemory(*maxMemory),
		storage.WithEvictionPolicy(policy),
	)

	if *sweepInterval > 0 {
		go func() {
			ticker := time.NewTicker(*sweepInterval)
			defer ticker.Stop()
			for range ticker.C {
				store.Sweep()
			}
		}()
	}

	config := server.ServerConfig{
		Address:         *address,
		UseRESPProtocol: true,
	}
	s := server.New(config)
	s.Serve()
}
