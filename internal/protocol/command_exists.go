package protocol

// HandleEXISTS processes the EXISTS command.
func HandleEXISTS(cmd *Command) []byte {
	store := commandStore()
	if len(cmd.Args) < 1 {
		return EncodeWrongArity("exists")
	}

	count := store.Exists(cmd.Args...)
	return EncodeInteger(int64(count))
}
