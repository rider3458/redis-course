package protocol

import (
	"strings"
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestDispatchCommand(t *testing.T) {
	restore := replaceCommandStoreForTest(storage.New())
	t.Cleanup(restore)

	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{name: "ping", cmd: &Command{Cmd: "PING"}, want: "+PONG\r\n"},
		{name: "set", cmd: &Command{Cmd: "SET", Args: []string{"k", "v"}}, want: "+OK\r\n"},
		{name: "get after set", cmd: &Command{Cmd: "GET", Args: []string{"k"}}, want: "$1\r\nv\r\n"},
		{name: "exists", cmd: &Command{Cmd: "EXISTS", Args: []string{"k"}}, want: ":1\r\n"},
		{name: "del", cmd: &Command{Cmd: "DEL", Args: []string{"k"}}, want: ":1\r\n"},
		{name: "unknown command", cmd: &Command{Cmd: "BOGUS"}, want: "-ERR unknown command 'BOGUS'\r\n"},
		{name: "nil command", cmd: nil, want: "-ERR internal error\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(DispatchCommand(tt.cmd)); got != tt.want {
				t.Fatalf("DispatchCommand = %q, want %q", got, tt.want)
			}
		})
	}

	if got := string(DispatchCommand(&Command{Cmd: "INFO", Args: []string{"stats"}})); !strings.Contains(got, "# Stats") {
		t.Fatalf("expected INFO to route to handler, got %q", got)
	}
}
