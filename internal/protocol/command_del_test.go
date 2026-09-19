package protocol

import (
	"testing"

	"github.com/rider3458/redis-course/internal/storage"
)

func TestHandleDEL(t *testing.T) {
	store := storage.New()
	restore := replaceCommandStoreForTest(store)
	t.Cleanup(restore)

	store.Set("a", "1", 0)
	store.Set("b", "2", 0)

	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "no args returns wrong arity",
			cmd:  &Command{Cmd: "DEL"},
			want: "-ERR wrong number of arguments for 'del' command\r\n",
		},
		{
			name: "missing key returns zero",
			cmd:  &Command{Cmd: "DEL", Args: []string{"missing"}},
			want: ":0\r\n",
		},
		{
			name: "existing key returns one",
			cmd:  &Command{Cmd: "DEL", Args: []string{"a"}},
			want: ":1\r\n",
		},
		{
			name: "mixed keys return count of deleted",
			cmd:  &Command{Cmd: "DEL", Args: []string{"missing", "b"}},
			want: ":1\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(HandleDEL(tt.cmd)); got != tt.want {
				t.Fatalf("HandleDEL = %q, want %q", got, tt.want)
			}
		})
	}
}
