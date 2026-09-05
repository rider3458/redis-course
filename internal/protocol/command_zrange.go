package protocol

import (
	"strconv"

	"github.com/rider3458/redis-course/internal/storage"
)

// HandleZRANGE processes the ZRANGE command.
func HandleZRANGE(cmd *Command) []byte {
	if len(cmd.Args) != 3 {
		return EncodeWrongArity("zrange")
	}

	start, err := strconv.Atoi(cmd.Args[1])
	if err != nil {
		return EncodeNotInteger()
	}
	stop, err := strconv.Atoi(cmd.Args[2])
	if err != nil {
		return EncodeNotInteger()
	}

	members, err := commandStore().ZRange(cmd.Args[0], start, stop)
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}
	return EncodeArrayBulkStrings(members)
}
