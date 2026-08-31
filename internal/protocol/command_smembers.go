package protocol

import "github.com/rider3458/redis-course/internal/storage"

// HandleSMEMBERS processes the SMEMBERS command.
func HandleSMEMBERS(cmd *Command) []byte {
	if len(cmd.Args) != 1 {
		return EncodeWrongArity("smembers")
	}

	members, err := commandStore().SMembers(cmd.Args[0])
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}

	return EncodeArrayBulkStrings(members)
}
