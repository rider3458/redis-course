package protocol

import "github.com/rider3458/redis-course/internal/storage"

// HandleSREM processes the SREM command.
func HandleSREM(cmd *Command) []byte {
	if len(cmd.Args) < 2 {
		return EncodeWrongArity("srem")
	}

	removed, err := commandStore().SRem(cmd.Args[0], cmd.Args[1:]...)
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}

	return EncodeInteger(int64(removed))
}
