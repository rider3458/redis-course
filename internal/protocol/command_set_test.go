package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleSET(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "no args returns wrong arity",
			cmd:  &Command{Cmd: "SET"},
			want: "-ERR wrong number of arguments for 'set' command\r\n",
		},
		{
			name: "one arg returns wrong arity",
			cmd:  &Command{Cmd: "SET", Args: []string{"k"}},
			want: "-ERR wrong number of arguments for 'set' command\r\n",
		},
		{
			name: "three args returns wrong arity",
			cmd:  &Command{Cmd: "SET", Args: []string{"k", "v", "extra"}},
			want: "-ERR wrong number of arguments for 'set' command\r\n",
		},
		{
			name: "set key value returns OK",
			cmd:  &Command{Cmd: "SET", Args: []string{"k", "v"}},
			want: "+OK\r\n",
		},
		{
			name: "set with EX seconds returns OK",
			cmd:  &Command{Cmd: "SET", Args: []string{"k", "v", "EX", "10"}},
			want: "+OK\r\n",
		},
		{
			name: "set with unknown option returns syntax error",
			cmd:  &Command{Cmd: "SET", Args: []string{"k", "v", "XX", "10"}},
			want: "-ERR syntax error\r\n",
		},
		{
			name: "set with non-numeric EX returns invalid expire",
			cmd:  &Command{Cmd: "SET", Args: []string{"k", "v", "EX", "abc"}},
			want: "-ERR invalid expire time in 'set' command\r\n",
		},
		{
			name: "set with zero EX returns invalid expire",
			cmd:  &Command{Cmd: "SET", Args: []string{"k", "v", "EX", "0"}},
			want: "-ERR invalid expire time in 'set' command\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(HandleSET(tt.cmd)); got != tt.want {
				t.Fatalf("HandleSET = %q, want %q", got, tt.want)
			}
		})
	}

	if got, want := string(HandleGET(&Command{Cmd: "GET", Args: []string{"k"}})), "$1\r\nv\r\n"; got != want {
		t.Fatalf("GET after SET = %q, want %q", got, want)
	}
	if ttl := commandStore().TTL("k"); ttl < 1 || ttl > 10 {
		t.Fatalf("unexpected TTL after SET EX: got=%d", ttl)
	}
}
