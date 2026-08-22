package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleEXISTS(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*storage.Store)
		cmd   *Command
		want  string
	}{
		{
			name: "wrong arity",
			cmd:  &Command{Cmd: "EXISTS", Args: []string{}},
			want: "-ERR wrong number of arguments for 'exists' command\r\n",
		},
		{
			name: "missing key returns zero",
			cmd:  &Command{Cmd: "EXISTS", Args: []string{"missing"}},
			want: ":0\r\n",
		},
		{
			name: "counts existing keys",
			setup: func(s *storage.Store) {
				s.Set("a", "1", 0)
				s.Set("b", "2", 0)
			},
			cmd:  &Command{Cmd: "EXISTS", Args: []string{"a", "x", "b"}},
			want: ":2\r\n",
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

			got := HandleEXISTS(tt.cmd)
			if string(got) != tt.want {
				t.Fatalf("unexpected response: got=%q want=%q", string(got), tt.want)
			}
		})
	}
}
