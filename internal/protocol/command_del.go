package protocol

// HandleDEL processes the DEL command.
func HandleDEL(cmd *Command) []byte {
	store := commandStore()
	argsLen := len(cmd.Args)

	if argsLen < 1 {
		return EncodeWrongArity("del")
	}

	deletedCount := store.Delete(cmd.Args...)
	return EncodeInteger(int64(deletedCount))
}
