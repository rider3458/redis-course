package protocol

import (
	"strconv"
	"strings"
	"time"
)

// HandleSET processes the SET command.
func HandleSET(cmd *Command) []byte {
	store := commandStore()
	argsLen := len(cmd.Args)
	if argsLen != 2 && argsLen != 4 {
		return EncodeWrongArity("set")
	}

	key := cmd.Args[0]
	value := cmd.Args[1]
	var ttl time.Duration

	if argsLen == 4 {
		if strings.ToUpper(cmd.Args[2]) != "EX" {
			return EncodeError("ERR syntax error")
		}

		seconds, err := strconv.ParseInt(cmd.Args[3], 10, 64)
		if err != nil || seconds <= 0 {
			return EncodeError("ERR invalid expire time in 'set' command")
		}

		ttl = time.Duration(seconds) * time.Second
	}

	store.Set(key, value, ttl)
	return EncodeSimpleString("OK")
}
