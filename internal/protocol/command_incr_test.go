package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleINCR(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*storage.Store)
		cmd   *Command
		want  string
	}{
		{
			name: "wrong arity",
			cmd:  &Command{Cmd: "INCR", Args: []string{}},
			want: "-ERR wrong number of arguments for 'incr' command\r\n",
		},
		{
			name: "missing key increments from one",
			cmd:  &Command{Cmd: "INCR", Args: []string{"n"}},
			want: ":1\r\n",
		},
		{
			name: "existing integer increments",
			setup: func(s *storage.Store) {
				s.Set("n", "10", 0)
			},
			cmd:  &Command{Cmd: "INCR", Args: []string{"n"}},
			want: ":11\r\n",
		},
		{
			name: "non integer value returns error",
			setup: func(s *storage.Store) {
				s.Set("n", "abc", 0)
			},
			cmd:  &Command{Cmd: "INCR", Args: []string{"n"}},
			want: "-ERR value is not an integer or out of range\r\n",
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

			got := HandleINCR(tt.cmd)
			if string(got) != tt.want {
				t.Fatalf("unexpected response: got=%q want=%q", string(got), tt.want)
			}
		})
	}
}
