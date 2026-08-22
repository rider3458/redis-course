package protocol

import (
	"testing"
	"time"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleEXPIRE(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*storage.Store)
		cmd   *Command
		want  string
	}{
		{
			name: "wrong arity",
			cmd:  &Command{Cmd: "EXPIRE", Args: []string{"k"}},
			want: "-ERR wrong number of arguments for 'expire' command\r\n",
		},
		{
			name: "invalid integer",
			cmd:  &Command{Cmd: "EXPIRE", Args: []string{"k", "abc"}},
			want: "-ERR value is not an integer or out of range\r\n",
		},
		{
			name: "missing key returns zero",
			cmd:  &Command{Cmd: "EXPIRE", Args: []string{"missing", "10"}},
			want: ":0\r\n",
		},
		{
			name: "existing key returns one",
			setup: func(s *storage.Store) {
				s.Set("k", "v", 0)
			},
			cmd:  &Command{Cmd: "EXPIRE", Args: []string{"k", "10"}},
			want: ":1\r\n",
		},
		{
			name: "non positive expires key",
			setup: func(s *storage.Store) {
				s.Set("k", "v", 0)
			},
			cmd:  &Command{Cmd: "EXPIRE", Args: []string{"k", "0"}},
			want: ":1\r\n",
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

			got := HandleEXPIRE(tt.cmd)
			if string(got) != tt.want {
				t.Fatalf("unexpected response: got=%q want=%q", string(got), tt.want)
			}

			if tt.name == "non positive expires key" {
				if _, ok := s.Get("k"); ok {
					t.Fatal("expected key to be deleted")
				}
			}
		})
	}

	_ = time.Second
}
