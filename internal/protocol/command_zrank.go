package protocol

import "github.com/rider3458/redis-course/internal/storage"

// HandleZRANK processes the ZRANK command.
func HandleZRANK(cmd *Command) []byte {
	if len(cmd.Args) != 2 {
		return EncodeWrongArity("zrank")
	}

	rank, found, err := commandStore().ZRank(cmd.Args[0], cmd.Args[1])
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}
	if !found {
		return EncodeNullBulkString()
	}
	return EncodeInteger(int64(rank))
}
