package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rider3458/redis-course/internal/protocol"
	server "github.com/rider3458/redis-course/internal/server"
	"github.com/rider3458/redis-course/internal/storage"
)

func main() {
	address := flag.String("addr", ":8080", "listen address")
	maxEntries := flag.Int("maxentries", 0, "evict when live keys exceed this count (0 = unlimited)")
	maxMemory := flag.Int64("maxmemory", 0, "evict when approximate memory exceeds this many bytes (0 = unlimited)")
	policyName := flag.String("maxmemory-policy", "lru", "eviction policy: lru, random, lfu, or noeviction")
	flag.Parse()

	policy, ok := storage.EvictionPolicyByName(*policyName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown -maxmemory-policy %q\n", *policyName)
		os.Exit(1)
	}

	protocol.ConfigureCommandStore(
		storage.WithMaxEntries(*maxEntries),
		storage.WithMaxMemory(*maxMemory),
		storage.WithEvictionPolicy(policy),
	)

	config := server.ServerConfig{
		Address:         *address,
		UseRESPProtocol: true,
	}
	s := server.New(config)
	s.Serve()
}
