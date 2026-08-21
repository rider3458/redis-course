package protocol

import (
	"errors"
	"reflect"
	"testing"
)

func TestDecodeOne_RESP2Types(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		wantValue    interface{}
		wantConsumed int
	}{
		{
			name:         "simple string",
			input:        []byte("+OK\r\n"),
			wantValue:    "OK",
			wantConsumed: len("+OK\r\n"),
		},
		{
			name:         "error",
			input:        []byte("-ERR bad thing\r\n"),
			wantValue:    RESPError("ERR bad thing"),
			wantConsumed: len("-ERR bad thing\r\n"),
		},
		{
			name:         "integer",
			input:        []byte(":-42\r\n"),
			wantValue:    int64(-42),
			wantConsumed: len(":-42\r\n"),
		},
		{
			name:         "bulk string",
			input:        []byte("$5\r\nhello\r\n"),
			wantValue:    "hello",
			wantConsumed: len("$5\r\nhello\r\n"),
		},
		{
			name:         "null bulk string",
			input:        []byte("$-1\r\n"),
			wantValue:    nil,
			wantConsumed: len("$-1\r\n"),
		},
		{
			name:         "array with mixed scalar values",
			input:        []byte("*3\r\n+OK\r\n:7\r\n$3\r\nhey\r\n"),
			wantValue:    []interface{}{"OK", int64(7), "hey"},
			wantConsumed: len("*3\r\n+OK\r\n:7\r\n$3\r\nhey\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, consumed, err := DecodeOne(tt.input)
			if err != nil {
				t.Fatalf("DecodeOne error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.wantValue) {
				t.Fatalf("value mismatch: got %#v want %#v", got, tt.wantValue)
			}

			if consumed != tt.wantConsumed {
				t.Fatalf("consumed mismatch: got %d want %d", consumed, tt.wantConsumed)
			}
		})
	}
}

func TestDecodeOne_Incomplete(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "empty", input: []byte{}},
		{name: "simple string truncated", input: []byte("+OK\r")},
		{name: "integer truncated", input: []byte(":12\r")},
		{name: "bulk length truncated", input: []byte("$5\r")},
		{name: "bulk payload truncated", input: []byte("$5\r\nhel")},
		{name: "array element truncated", input: []byte("*1\r\n$4\r\nPI")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DecodeOne(tt.input)
			if !errors.Is(err, ErrIncompleteRESP) {
				t.Fatalf("expected ErrIncompleteRESP, got %v", err)
			}
		})
	}
}

func TestParseCommand_ValidCases(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		wantCmd      string
		wantArgs     []string
		wantConsumed int
	}{
		{
			name:         "single token command",
			input:        []byte("*1\r\n$4\r\nPING\r\n"),
			wantCmd:      "PING",
			wantArgs:     []string{},
			wantConsumed: len("*1\r\n$4\r\nPING\r\n"),
		},
		{
			name:         "command with one arg",
			input:        []byte("*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n"),
			wantCmd:      "GET",
			wantArgs:     []string{"key"},
			wantConsumed: len("*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n"),
		},
		{
			name:         "command is uppercased",
			input:        []byte("*2\r\n$4\r\nping\r\n$4\r\nPONG\r\n"),
			wantCmd:      "PING",
			wantArgs:     []string{"PONG"},
			wantConsumed: len("*2\r\n$4\r\nping\r\n$4\r\nPONG\r\n"),
		},
		{
			name:         "set with two args",
			input:        []byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"),
			wantCmd:      "SET",
			wantArgs:     []string{"key", "value"},
			wantConsumed: len("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"),
		},
		{
			name:         "supports zero-length bulk string arg",
			input:        []byte("*2\r\n$3\r\nGET\r\n$0\r\n\r\n"),
			wantCmd:      "GET",
			wantArgs:     []string{""},
			wantConsumed: len("*2\r\n$3\r\nGET\r\n$0\r\n\r\n"),
		},
		{
			name:         "returns consumed bytes for concatenated commands",
			input:        []byte("*1\r\n$4\r\nPING\r\n*1\r\n$4\r\nPONG\r\n"),
			wantCmd:      "PING",
			wantArgs:     []string{},
			wantConsumed: len("*1\r\n$4\r\nPING\r\n"),
		},
		{
			name:         "integer args are stringified",
			input:        []byte("*2\r\n$4\r\nINCR\r\n:5\r\n"),
			wantCmd:      "INCR",
			wantArgs:     []string{"5"},
			wantConsumed: len("*2\r\n$4\r\nINCR\r\n:5\r\n"),
		},
		{
			name:         "simple string tokens in array",
			input:        []byte("*2\r\n+PING\r\n+PONG\r\n"),
			wantCmd:      "PING",
			wantArgs:     []string{"PONG"},
			wantConsumed: len("*2\r\n+PING\r\n+PONG\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, consumed, err := ParseCommand(tt.input)
			if err != nil {
				t.Fatalf("ParseCommand error = %v", err)
			}

			if got.Cmd != tt.wantCmd {
				t.Fatalf("Cmd mismatch: got %q want %q", got.Cmd, tt.wantCmd)
			}

			if !reflect.DeepEqual(got.Args, tt.wantArgs) {
				t.Fatalf("Args mismatch: got %#v want %#v", got.Args, tt.wantArgs)
			}

			if consumed != tt.wantConsumed {
				t.Fatalf("consumed mismatch: got %d want %d", consumed, tt.wantConsumed)
			}
		})
	}
}

func TestParseCommand_IncompleteCases(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "empty input", input: []byte{}},
		{name: "truncated array length", input: []byte("*2\r")},
		{name: "truncated bulk length", input: []byte("*1\r\n$4\r")},
		{name: "missing bulk payload", input: []byte("*1\r\n$4\r\nPI")},
		{name: "missing final crlf", input: []byte("*1\r\n$4\r\nPING\r")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseCommand(tt.input)
			if !errors.Is(err, ErrIncompleteRESP) {
				t.Fatalf("expected ErrIncompleteRESP, got %v", err)
			}
		})
	}
}

func TestParseCommand_InvalidCases(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "wrong top-level prefix", input: []byte("+OK\r\n")},
		{name: "invalid arity zero", input: []byte("*0\r\n")},
		{name: "invalid length token", input: []byte("*x\r\n")},
		{name: "nil bulk string token", input: []byte("*1\r\n$-1\r\n")},
		{name: "bulk terminator not crlf", input: []byte("*1\r\n$4\r\nPING\n")},
		{name: "nested array token not allowed in command", input: []byte("*2\r\n$4\r\nPING\r\n*1\r\n$1\r\na\r\n")},
		{name: "error token not allowed in command", input: []byte("*2\r\n$4\r\nPING\r\n-ERR nope\r\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseCommand(tt.input)
			if err == nil {
				t.Fatalf("expected error for input %q", string(tt.input))
			}
		})
	}
}

func TestDecode(t *testing.T) {
	t.Run("decode exact value", func(t *testing.T) {
		value, err := Decode([]byte("+OK\r\n"))
		if err != nil {
			t.Fatalf("Decode error = %v", err)
		}
		if value != "OK" {
			t.Fatalf("unexpected value: %#v", value)
		}
	})

	t.Run("fails on trailing bytes", func(t *testing.T) {
		_, err := Decode([]byte("+OK\r\nextra"))
		if err == nil {
			t.Fatal("expected trailing bytes error")
		}
	})
}

func TestEncodeHelpers(t *testing.T) {
	t.Run("EncodeSimpleString", func(t *testing.T) {
		got := EncodeSimpleString("PONG")
		want := []byte("+PONG\r\n")
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("EncodeSimpleString mismatch: got %q want %q", string(got), string(want))
		}
	})

	t.Run("EncodeError", func(t *testing.T) {
		got := EncodeError("ERR nope")
		want := []byte("-ERR nope\r\n")
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("EncodeError mismatch: got %q want %q", string(got), string(want))
		}
	})
}

func TestParseCommandExact(t *testing.T) {
	t.Run("exact success", func(t *testing.T) {
		input := []byte("*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n")
		cmd, err := ParseCommandExact(input)
		if err != nil {
			t.Fatalf("ParseCommandExact error = %v", err)
		}
		if cmd.Cmd != "GET" || !reflect.DeepEqual(cmd.Args, []string{"key"}) {
			t.Fatalf("unexpected command parsed: %#v", cmd)
		}
	})

	t.Run("fails with trailing bytes", func(t *testing.T) {
		input := []byte("*1\r\n$4\r\nPING\r\nextra")
		_, err := ParseCommandExact(input)
		if err == nil {
			t.Fatal("expected trailing bytes error")
		}
	})
}
