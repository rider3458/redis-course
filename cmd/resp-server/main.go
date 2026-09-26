package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
	shutdownTimeout := flag.Duration("shutdown-timeout", 5*time.Second, "how long to let in-flight commands finish before closing connections")
	flag.Parse()

	policy, ok := storage.EvictionPolicyByName(*policyName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown -maxmemory-policy %q\n", *policyName)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := protocol.ConfigureCommandStore(
		storage.WithMaxEntries(*maxEntries),
		storage.WithMaxMemory(*maxMemory),
		storage.WithEvictionPolicy(policy),
	)

	if *sweepInterval > 0 {
		go func() {
			ticker := time.NewTicker(*sweepInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					store.Sweep()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	config := server.ServerConfig{
		Address:         *address,
		UseRESPProtocol: true,
		ShutdownTimeout: *shutdownTimeout,
	}
	s := server.New(config)
	if err := s.ListenAndServe(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
