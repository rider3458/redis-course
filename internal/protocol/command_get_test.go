package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleGET(t *testing.T) {
	store := storage.New()
	restore := replaceCommandStoreForTest(store)
	t.Cleanup(restore)

	store.Set("k", "v", 0)
	if _, err := store.SAdd("set", "m"); err != nil {
		t.Fatalf("unexpected SAdd error: %v", err)
	}

	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "no args returns wrong arity",
			cmd:  &Command{Cmd: "GET"},
			want: "-ERR wrong number of arguments for 'get' command\r\n",
		},
		{
			name: "two args returns wrong arity",
			cmd:  &Command{Cmd: "GET", Args: []string{"a", "b"}},
			want: "-ERR wrong number of arguments for 'get' command\r\n",
		},
		{
			name: "missing key returns null bulk string",
			cmd:  &Command{Cmd: "GET", Args: []string{"missing"}},
			want: "$-1\r\n",
		},
		{
			name: "existing key returns bulk string",
			cmd:  &Command{Cmd: "GET", Args: []string{"k"}},
			want: "$1\r\nv\r\n",
		},
		{
			name: "wrong type key returns null bulk string",
			cmd:  &Command{Cmd: "GET", Args: []string{"set"}},
			want: "$-1\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(HandleGET(tt.cmd)); got != tt.want {
				t.Fatalf("HandleGET = %q, want %q", got, tt.want)
			}
		})
	}
}
