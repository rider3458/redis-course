package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestBloomFilterCommands(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	if got, want := DispatchCommand(&Command{Cmd: "BF.RESERVE", Args: []string{"seen", "0.01", "100"}}), "+OK\r\n"; string(got) != want {
		t.Fatalf("BF.RESERVE = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "BF.ADD", Args: []string{"seen", "home"}}), ":1\r\n"; string(got) != want {
		t.Fatalf("BF.ADD = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "BF.ADD", Args: []string{"seen", "home"}}), ":0\r\n"; string(got) != want {
		t.Fatalf("duplicate BF.ADD = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "BF.MADD", Args: []string{"seen", "about", "contact"}}), "*2\r\n:1\r\n:1\r\n"; string(got) != want {
		t.Fatalf("BF.MADD = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "BF.MEXISTS", Args: []string{"seen", "home", "about", "missing"}}), "*3\r\n:1\r\n:1\r\n:0\r\n"; string(got) != want {
		t.Fatalf("BF.MEXISTS = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "BF.EXISTS", Args: []string{"missing", "item"}}), ":0\r\n"; string(got) != want {
		t.Fatalf("missing BF.EXISTS = %q, want %q", got, want)
	}
	if got, want := DispatchCommand(&Command{Cmd: "BF.INFO", Args: []string{"seen"}}), "*10\r\n$8\r\nCapacity\r\n:100\r\n$4\r\nSize\r\n:120\r\n$17\r\nNumber of filters\r\n:1\r\n$24\r\nNumber of items inserted\r\n:3\r\n$14\r\nExpansion rate\r\n:0\r\n"; string(got) != want {
		t.Fatalf("BF.INFO = %q, want %q", got, want)
	}
}

func TestBloomFilterCommandsRejectInvalidInputAndTypes(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	if got, want := HandleBFReserve(&Command{Args: []string{"seen", "0", "100"}}), "-ERR BF: error rate must be between 0 and 1\r\n"; string(got) != want {
		t.Fatalf("invalid error rate = %q, want %q", got, want)
	}
	if got, want := HandleBFReserve(&Command{Args: []string{"seen", "0.01", "0"}}), "-ERR BF: capacity must be a positive integer\r\n"; string(got) != want {
		t.Fatalf("invalid capacity = %q, want %q", got, want)
	}
	commandStore().Set("string", "value", 0)
	if got, want := HandleBFAdd(&Command{Args: []string{"string", "item"}}), "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"; string(got) != want {
		t.Fatalf("wrong type = %q, want %q", got, want)
	}
}
