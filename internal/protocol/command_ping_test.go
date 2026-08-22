package protocol

import "testing"

func TestHandlePING(t *testing.T) {
	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "no args returns PONG",
			cmd:  &Command{Cmd: "PING", Args: []string{}},
			want: "+PONG\r\n",
		},
		{
			name: "one arg echoes message",
			cmd:  &Command{Cmd: "PING", Args: []string{"HELLO"}},
			want: "+HELLO\r\n",
		},
		{
			name: "too many args returns wrong arity",
			cmd:  &Command{Cmd: "PING", Args: []string{"a", "b"}},
			want: "-ERR wrong number of arguments for 'ping' command\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HandlePING(tt.cmd)
			if string(got) != tt.want {
				t.Fatalf("unexpected response: got=%q want=%q", string(got), tt.want)
			}
		})
	}
}
