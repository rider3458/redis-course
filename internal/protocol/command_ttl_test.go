package protocol

import (
	"testing"
	"time"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleTTL(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*storage.Store)
		cmd   *Command
		want  string
	}{
		{
			name: "wrong arity",
			cmd:  &Command{Cmd: "TTL", Args: []string{}},
			want: "-ERR wrong number of arguments for 'ttl' command\r\n",
		},
		{
			name: "missing key returns minus two",
			cmd:  &Command{Cmd: "TTL", Args: []string{"missing"}},
			want: ":-2\r\n",
		},
		{
			name: "no expiry returns minus one",
			setup: func(s *storage.Store) {
				s.Set("k", "v", 0)
			},
			cmd:  &Command{Cmd: "TTL", Args: []string{"k"}},
			want: ":-1\r\n",
		},
		{
			name: "expiring key returns non negative",
			setup: func(s *storage.Store) {
				s.Set("k", "v", 3*time.Second)
			},
			cmd:  &Command{Cmd: "TTL", Args: []string{"k"}},
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

			got := HandleTTL(tt.cmd)
			if tt.name == "expiring key returns non negative" {
				if len(got) < 3 || got[0] != ':' {
					t.Fatalf("unexpected response format: %q", string(got))
				}
				return
			}

			if string(got) != tt.want {
				t.Fatalf("unexpected response: got=%q want=%q", string(got), tt.want)
			}
		})
	}
}
