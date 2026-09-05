package protocol

import (
	"strconv"

	"github.com/rider3458/redis-course/internal/storage"
)

// HandleZSCORE processes the ZSCORE command.
func HandleZSCORE(cmd *Command) []byte {
	if len(cmd.Args) != 2 {
		return EncodeWrongArity("zscore")
	}

	score, found, err := commandStore().ZScore(cmd.Args[0], cmd.Args[1])
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}
	if !found {
		return EncodeNullBulkString()
	}
	return EncodeBulkString(strconv.FormatFloat(score, 'f', -1, 64))
}
