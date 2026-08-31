package protocol

import "github.com/rider3458/redis-course/internal/storage"

// HandleSADD processes the SADD command.
func HandleSADD(cmd *Command) []byte {
	if len(cmd.Args) < 2 {
		return EncodeWrongArity("sadd")
	}

	added, err := commandStore().SAdd(cmd.Args[0], cmd.Args[1:]...)
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}

	return EncodeInteger(int64(added))
}
