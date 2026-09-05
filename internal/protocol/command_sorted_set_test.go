package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestSortedSetCommands(t *testing.T) {
	store := storage.New()
	restore := replaceCommandStoreForTest(store)
	defer restore()

	assertResponse(t, HandleZADD(&Command{Cmd: "ZADD", Args: []string{"scores", "2.5", "alice", "1", "bob", "2.5", "carol"}}), ":3\r\n")
	assertResponse(t, HandleZSCORE(&Command{Cmd: "ZSCORE", Args: []string{"scores", "alice"}}), "$3\r\n2.5\r\n")
	assertResponse(t, HandleZRANK(&Command{Cmd: "ZRANK", Args: []string{"scores", "bob"}}), ":0\r\n")
	assertResponse(t, HandleZRANK(&Command{Cmd: "ZRANK", Args: []string{"scores", "carol"}}), ":2\r\n")
	assertResponse(t, HandleZRANGE(&Command{Cmd: "ZRANGE", Args: []string{"scores", "0", "1"}}), "*2\r\n$3\r\nbob\r\n$5\r\nalice\r\n")
	assertResponse(t, HandleZRANGE(&Command{Cmd: "ZRANGE", Args: []string{"scores", "-2", "-1"}}), "*2\r\n$5\r\nalice\r\n$5\r\ncarol\r\n")
	assertResponse(t, HandleZADD(&Command{Cmd: "ZADD", Args: []string{"scores", "0", "alice"}}), ":0\r\n")
	assertResponse(t, HandleZRANK(&Command{Cmd: "ZRANK", Args: []string{"scores", "alice"}}), ":0\r\n")
	assertResponse(t, HandleZREM(&Command{Cmd: "ZREM", Args: []string{"scores", "alice", "missing"}}), ":1\r\n")
	assertResponse(t, HandleZSCORE(&Command{Cmd: "ZSCORE", Args: []string{"scores", "alice"}}), "$-1\r\n")
}

func TestSortedSetCommandErrors(t *testing.T) {
	assertResponse(t, HandleZADD(&Command{Cmd: "ZADD", Args: []string{"scores", "not-a-score", "member"}}), "-ERR value is not a valid float\r\n")
	assertResponse(t, HandleZADD(&Command{Cmd: "ZADD", Args: []string{"scores", "NaN", "member"}}), "-ERR value is not a valid float\r\n")
	assertResponse(t, HandleZADD(&Command{Cmd: "ZADD", Args: []string{"scores", "1"}}), "-ERR wrong number of arguments for 'zadd' command\r\n")
	assertResponse(t, HandleZSCORE(&Command{Cmd: "ZSCORE", Args: []string{"scores"}}), "-ERR wrong number of arguments for 'zscore' command\r\n")
	assertResponse(t, HandleZRANK(&Command{Cmd: "ZRANK", Args: []string{"scores"}}), "-ERR wrong number of arguments for 'zrank' command\r\n")
	assertResponse(t, HandleZRANGE(&Command{Cmd: "ZRANGE", Args: []string{"scores", "0"}}), "-ERR wrong number of arguments for 'zrange' command\r\n")
	assertResponse(t, HandleZRANGE(&Command{Cmd: "ZRANGE", Args: []string{"scores", "zero", "1"}}), "-ERR value is not an integer or out of range\r\n")
	assertResponse(t, HandleZREM(&Command{Cmd: "ZREM", Args: []string{"scores"}}), "-ERR wrong number of arguments for 'zrem' command\r\n")

	store := storage.New()
	store.Set("string", "value", 0)
	restore := replaceCommandStoreForTest(store)
	defer restore()
	assertResponse(t, HandleZSCORE(&Command{Cmd: "ZSCORE", Args: []string{"string", "member"}}), "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n")
}

func TestSortedSetCommandsDispatch(t *testing.T) {
	store := storage.New()
	restore := replaceCommandStoreForTest(store)
	defer restore()

	assertResponse(t, DispatchCommand(&Command{Cmd: "ZADD", Args: []string{"scores", "1", "member"}}), ":1\r\n")
	assertResponse(t, DispatchCommand(&Command{Cmd: "ZRANGE", Args: []string{"scores", "0", "-1"}}), "*1\r\n$6\r\nmember\r\n")
}

func assertResponse(t *testing.T, got []byte, want string) {
	t.Helper()
	if string(got) != want {
		t.Fatalf("unexpected response: got=%q want=%q", string(got), want)
	}
}
