package protocol

import "testing"

func TestCommandErrorEncoders(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want string
	}{
		{
			name: "unknown command",
			got:  EncodeUnknownCommand("FOO"),
			want: "-ERR unknown command 'FOO'\r\n",
		},
		{
			name: "wrong arity",
			got:  EncodeWrongArity("set"),
			want: "-ERR wrong number of arguments for 'set' command\r\n",
		},
		{
			name: "not integer",
			got:  EncodeNotInteger(),
			want: "-ERR value is not an integer or out of range\r\n",
		},
		{
			name: "wrong type",
			got:  EncodeWrongType(),
			want: "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.want {
				t.Fatalf("encoder = %q, want %q", string(tt.got), tt.want)
			}
		})
	}
}
