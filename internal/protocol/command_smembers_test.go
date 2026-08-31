package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleSMEMBERS(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*storage.Store)
		cmd   *Command
		want  string
	}{
		{
			name: "wrong arity",
			cmd:  &Command{Cmd: "SMEMBERS", Args: []string{"k", "x"}},
			want: "-ERR wrong number of arguments for 'smembers' command\r\n",
		},
		{
			name: "missing key returns empty array",
			cmd:  &Command{Cmd: "SMEMBERS", Args: []string{"missing"}},
			want: "*0\r\n",
		},
		{
			name: "returns sorted members",
			setup: func(s *storage.Store) {
				_, _ = s.SAdd("k", "b", "a")
			},
			cmd:  &Command{Cmd: "SMEMBERS", Args: []string{"k"}},
			want: "*2\r\n$1\r\na\r\n$1\r\nb\r\n",
		},
		{
			name: "wrong type",
			setup: func(s *storage.Store) {
				s.Set("k", "v", 0)
			},
			cmd:  &Command{Cmd: "SMEMBERS", Args: []string{"k"}},
			want: "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := storage.New()
			restore := replaceCommandStoreForTest(s)
			defer restore()
			if tt.setup != nil {
				tt.setup(s)
			}

			got := HandleSMEMBERS(tt.cmd)
			if string(got) != tt.want {
				t.Fatalf("unexpected response: got=%q want=%q", string(got), tt.want)
			}
		})
	}
}
