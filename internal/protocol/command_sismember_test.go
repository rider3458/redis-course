package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleSISMEMBER(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*storage.Store)
		cmd   *Command
		want  string
	}{
		{
			name: "wrong arity",
			cmd:  &Command{Cmd: "SISMEMBER", Args: []string{"k"}},
			want: "-ERR wrong number of arguments for 'sismember' command\r\n",
		},
		{
			name: "member exists",
			setup: func(s *storage.Store) {
				_, _ = s.SAdd("k", "a", "b")
			},
			cmd:  &Command{Cmd: "SISMEMBER", Args: []string{"k", "a"}},
			want: ":1\r\n",
		},
		{
			name: "member missing",
			setup: func(s *storage.Store) {
				_, _ = s.SAdd("k", "a", "b")
			},
			cmd:  &Command{Cmd: "SISMEMBER", Args: []string{"k", "x"}},
			want: ":0\r\n",
		},
		{
			name: "wrong type",
			setup: func(s *storage.Store) {
				s.Set("k", "v", 0)
			},
			cmd:  &Command{Cmd: "SISMEMBER", Args: []string{"k", "a"}},
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

			got := HandleSISMEMBER(tt.cmd)
			if string(got) != tt.want {
				t.Fatalf("unexpected response: got=%q want=%q", string(got), tt.want)
			}
		})
	}
}
