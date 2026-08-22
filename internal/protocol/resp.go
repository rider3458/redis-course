package protocol

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var ErrIncompleteRESP = errors.New("incomplete RESP message")

type RESPError string

func (e RESPError) Error() string {
	return string(e)
}

type Command struct {
	Cmd  string
	Args []string
}

// ParseCommand parses a single RESP command from data and returns the
// parsed command plus the number of bytes consumed.
func ParseCommand(data []byte) (*Command, int, error) {
	value, consumed, err := DecodeOne(data)
	if err != nil {
		return nil, 0, err
	}

	array, ok := value.([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("expected top-level RESP array, got %s", reflect.TypeOf(value))
	}

	if len(array) == 0 {
		return nil, 0, fmt.Errorf("invalid command arity: 0")
	}

	cmdToken, err := tokenToString(array[0], false)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid command token: %w", err)
	}

	args := make([]string, 0, len(array)-1)
	for i := 1; i < len(array); i++ {
		arg, err := tokenToString(array[i], true)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid argument token at index %d: %w", i, err)
		}
		args = append(args, arg)
	}

	cmd := &Command{
		Cmd:  strings.ToUpper(cmdToken),
		Args: args,
	}

	return cmd, consumed, nil
}

// DecodeOne decodes one RESP2 value from data and returns the parsed value,
// consumed byte count, and error.
//
// Supported value mappings:
// - Simple string (+): string
// - Error (-): RESPError
// - Integer (:): int64
// - Bulk string ($): string or nil for null bulk
// - Array (*): []interface{} or nil for null array
func DecodeOne(data []byte) (interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, ErrIncompleteRESP
	}

	switch data[0] {
	case '+':
		return readSimpleString(data)
	case '-':
		msg, consumed, err := readSimpleString(data)
		if err != nil {
			return nil, 0, err
		}
		return RESPError(msg), consumed, nil
	case ':':
		return readInteger(data)
	case '$':
		return readBulkString(data)
	case '*':
		return readArray(data)
	default:
		return nil, 0, fmt.Errorf("unknown RESP prefix %q", data[0])
	}
}

// Decode decodes one RESP2 value and requires full input consumption.
func Decode(data []byte) (interface{}, error) {
	value, consumed, err := DecodeOne(data)
	if err != nil {
		return nil, err
	}
	if consumed != len(data) {
		return nil, fmt.Errorf("extra bytes after value: consumed=%d total=%d", consumed, len(data))
	}
	return value, nil
}

func EncodeSimpleString(value string) []byte {
	return []byte("+" + value + "\r\n")
}

func EncodeBulkString(value string) []byte {
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(value), value))
}

func EncodeNullBulkString() []byte {
	return []byte("$-1\r\n")
}

func EncodeInteger(value int64) []byte {
	return []byte(":" + strconv.FormatInt(value, 10) + "\r\n")
}

func EncodeError(message string) []byte {
	return []byte("-" + message + "\r\n")
}

// ParseCommandExact parses a single RESP command and requires the full input
// to be consumed by that command.
func ParseCommandExact(data []byte) (*Command, error) {
	cmd, consumed, err := ParseCommand(data)
	if err != nil {
		return nil, err
	}
	if consumed != len(data) {
		return nil, fmt.Errorf("extra bytes after command: consumed=%d total=%d", consumed, len(data))
	}
	return cmd, nil
}

func tokenToString(value interface{}, allowInt bool) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int64:
		if allowInt {
			return strconv.FormatInt(v, 10), nil
		}
		return "", fmt.Errorf("integer token is not allowed here")
	case nil:
		return "", fmt.Errorf("null token is not allowed")
	default:
		return "", fmt.Errorf("unsupported token type %T", value)
	}
}

func readSimpleString(data []byte) (string, int, error) {
	line, nextPos, err := readLine(data, 1)
	if err != nil {
		return "", 0, err
	}
	return line, nextPos, nil
}

func readInteger(data []byte) (int64, int, error) {
	line, nextPos, err := readLine(data, 1)
	if err != nil {
		return 0, 0, err
	}

	n, convErr := strconv.ParseInt(line, 10, 64)
	if convErr != nil {
		return 0, 0, fmt.Errorf("invalid integer value %q", line)
	}

	return n, nextPos, nil
}

func readBulkString(data []byte) (interface{}, int, error) {
	length, nextPos, err := readLengthLine(data, 1)
	if err != nil {
		return nil, 0, err
	}

	if length == -1 {
		return nil, nextPos, nil
	}
	if length < -1 {
		return nil, 0, fmt.Errorf("invalid bulk string length: %d", length)
	}

	if len(data) < nextPos+length+2 {
		return nil, 0, ErrIncompleteRESP
	}

	if data[nextPos+length] != '\r' || data[nextPos+length+1] != '\n' {
		return nil, 0, fmt.Errorf("bulk string missing CRLF terminator")
	}

	return string(data[nextPos : nextPos+length]), nextPos + length + 2, nil
}

func readArray(data []byte) (interface{}, int, error) {
	length, nextPos, err := readLengthLine(data, 1)
	if err != nil {
		return nil, 0, err
	}

	if length == -1 {
		return nil, nextPos, nil
	}
	if length < -1 {
		return nil, 0, fmt.Errorf("invalid array length: %d", length)
	}

	result := make([]interface{}, 0, length)
	pos := nextPos
	for i := 0; i < length; i++ {
		value, consumed, err := DecodeOne(data[pos:])
		if err != nil {
			return nil, 0, err
		}
		result = append(result, value)
		pos += consumed
	}

	return result, pos, nil
}

func readLengthLine(data []byte, start int) (int, int, error) {
	line, nextPos, err := readLine(data, start)
	if err != nil {
		return 0, 0, err
	}

	n, convErr := strconv.Atoi(line)
	if convErr != nil {
		return 0, 0, fmt.Errorf("invalid length value %q", line)
	}

	return n, nextPos, nil
}

func readLine(data []byte, start int) (string, int, error) {
	for i := start; i+1 < len(data); i++ {
		if data[i] == '\r' && data[i+1] == '\n' {
			return string(data[start:i]), i + 2, nil
		}
	}
	return "", 0, ErrIncompleteRESP
}
