package protocol

import (
	"strconv"
	"time"
)

// HandleEXPIRE processes the EXPIRE command.
func HandleEXPIRE(cmd *Command) []byte {
	if len(cmd.Args) != 2 {
		return EncodeWrongArity("expire")
	}

	seconds, err := strconv.ParseInt(cmd.Args[1], 10, 64)
	if err != nil {
		return EncodeNotInteger()
	}

	ok := commandStore().Expire(cmd.Args[0], time.Duration(seconds)*time.Second)
	if ok {
		return EncodeInteger(1)
	}
	return EncodeInteger(0)
}
