package protocol

import "github.com/rider3458/redis-course/internal/storage"

// HandleSISMEMBER processes the SISMEMBER command.
func HandleSISMEMBER(cmd *Command) []byte {
	if len(cmd.Args) != 2 {
		return EncodeWrongArity("sismember")
	}

	isMember, err := commandStore().SIsMember(cmd.Args[0], cmd.Args[1])
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}

	return EncodeInteger(int64(isMember))
}
