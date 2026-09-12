package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestCMSCommands(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	if got, want := DispatchCommand(&Command{Cmd: "CMS.INITBYDIM", Args: []string{"visits", "100", "5"}}), "+OK\r\n"; string(got) != want {
		t.Fatalf("CMS.INITBYDIM = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "CMS.INCRBY", Args: []string{"visits", "home", "3", "about", "2", "home", "4"}}), "*3\r\n:3\r\n:2\r\n:7\r\n"; string(got) != want {
		t.Fatalf("CMS.INCRBY = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "CMS.QUERY", Args: []string{"visits", "home", "about", "missing"}}), "*3\r\n:7\r\n:2\r\n:0\r\n"; string(got) != want {
		t.Fatalf("CMS.QUERY = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "CMS.INFO", Args: []string{"visits"}}), "*6\r\n$5\r\nwidth\r\n:100\r\n$5\r\ndepth\r\n:5\r\n$5\r\ncount\r\n:9\r\n"; string(got) != want {
		t.Fatalf("CMS.INFO = %q, want %q", got, want)
	}
}

func TestCMSInitByProbability(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	if got, want := HandleCMSInitByProb(&Command{Args: []string{"events", "0.1", "0.01"}}), "+OK\r\n"; string(got) != want {
		t.Fatalf("CMS.INITBYPROB = %q, want %q", got, want)
	}
	if got, want := HandleCMSInfo(&Command{Args: []string{"events"}}), "*6\r\n$5\r\nwidth\r\n:28\r\n$5\r\ndepth\r\n:5\r\n$5\r\ncount\r\n:0\r\n"; string(got) != want {
		t.Fatalf("CMS.INFO = %q, want %q", got, want)
	}
}

func TestCMSMergeWithWeights(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)
	for _, key := range []string{"first", "second"} {
		if got := HandleCMSInitByDim(&Command{Args: []string{key, "100", "5"}}); string(got) != "+OK\r\n" {
			t.Fatalf("CMS.INITBYDIM = %q", got)
		}
	}
	HandleCMSIncrBy(&Command{Args: []string{"first", "home", "3"}})
	HandleCMSIncrBy(&Command{Args: []string{"second", "home", "2", "about", "4"}})

	if got, want := HandleCMSMerge(&Command{Args: []string{"combined", "2", "first", "second", "WEIGHTS", "2", "1"}}), "+OK\r\n"; string(got) != want {
		t.Fatalf("CMS.MERGE = %q, want %q", got, want)
	}
	if got, want := HandleCMSQuery(&Command{Args: []string{"combined", "home", "about"}}), "*2\r\n:8\r\n:4\r\n"; string(got) != want {
		t.Fatalf("CMS.QUERY = %q, want %q", got, want)
	}
}

func TestCMSCommandsRejectInvalidInputAndTypes(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)
	if got := HandleCMSInitByDim(&Command{Args: []string{"sketch", "0", "5"}}); string(got) != "-ERR CMS: width must be a positive integer\r\n" {
		t.Fatalf("invalid width response = %q", got)
	}
	if got := HandleCMSIncrBy(&Command{Args: []string{"sketch", "item", "0"}}); string(got) != "-ERR CMS: increment must be a positive integer\r\n" {
		t.Fatalf("invalid increment response = %q", got)
	}
	if got := HandleCMSQuery(&Command{Args: []string{"missing", "item"}}); string(got) != "-ERR CMS: key does not exist\r\n" {
		t.Fatalf("missing response = %q", got)
	}
	commandStore().Set("string", "value", 0)
	if got := HandleCMSQuery(&Command{Args: []string{"string", "item"}}); string(got) != "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n" {
		t.Fatalf("wrong type response = %q", got)
	}
}
