package protocol

import "github.com/rider3458/redis-course/internal/storage"

// HandleZREM processes the ZREM command.
func HandleZREM(cmd *Command) []byte {
	if len(cmd.Args) < 2 {
		return EncodeWrongArity("zrem")
	}

	removed, err := commandStore().ZRem(cmd.Args[0], cmd.Args[1:]...)
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}
	return EncodeInteger(int64(removed))
}
