package protocol

import (
	"fmt"
	"strings"

	"github.com/rider3458/redis-course/internal/storage"
)

// infoDefaultSections are rendered when INFO is called without a section.
var infoDefaultSections = []string{"memory", "stats", "keyspace"}

// HandleINFO processes the INFO command.
//
// Sections are optional and variadic: INFO [section [section ...]].
// Unknown sections are ignored.
func HandleINFO(cmd *Command) []byte {
	stats := commandStore().Stats()

	sections := infoDefaultSections
	if len(cmd.Args) > 0 {
		sections = requestedInfoSections(cmd.Args)
	}

	var builder strings.Builder
	for _, section := range sections {
		renderInfoSection(&builder, section, stats)
	}
	return EncodeBulkString(builder.String())
}

// requestedInfoSections maps INFO arguments to supported section names.
func requestedInfoSections(args []string) []string {
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "all", "everything":
			return []string{"memory", "stats", "keyspace"}
		case "default":
			return infoDefaultSections
		}
	}

	var sections []string
	for _, arg := range args {
		name := strings.ToLower(arg)
		if infoSectionSupported(name) {
			sections = append(sections, name)
		}
	}
	return sections
}

func infoSectionSupported(name string) bool {
	switch name {
	case "memory", "stats", "keyspace":
		return true
	default:
		return false
	}
}

func renderInfoSection(builder *strings.Builder, section string, stats storage.StoreStats) {
	switch section {
	case "memory":
		builder.WriteString("# Memory\r\n")
		fmt.Fprintf(builder, "used_memory:%d\r\n", stats.UsedMemory)
		fmt.Fprintf(builder, "maxmemory:%d\r\n", stats.MaxMemory)
		fmt.Fprintf(builder, "maxmemory_policy:%s\r\n", stats.EvictionPolicy)
	case "stats":
		builder.WriteString("# Stats\r\n")
		fmt.Fprintf(builder, "keyspace_hits:%d\r\n", stats.Hits)
		fmt.Fprintf(builder, "keyspace_misses:%d\r\n", stats.Misses)
		fmt.Fprintf(builder, "expired_keys:%d\r\n", stats.ExpiredKeys)
		fmt.Fprintf(builder, "evicted_keys:%d\r\n", stats.EvictedKeys)
	case "keyspace":
		builder.WriteString("# Keyspace\r\n")
		if stats.Keys > 0 {
			fmt.Fprintf(builder, "db0:keys=%d,expires=0,avg_ttl=0\r\n", stats.Keys)
		}
	}
	builder.WriteString("\r\n")
}
