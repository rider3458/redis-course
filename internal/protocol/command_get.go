package protocol

// HandleGET processes the GET command.
func HandleGET(cmd *Command) []byte {
	store := commandStore()
	argsLen := len(cmd.Args)

	if argsLen != 1 {
		return EncodeWrongArity("get")
	}

	value, ok := store.Get(cmd.Args[0])
	if !ok {
		return EncodeNullBulkString()
	}

	return EncodeBulkString(value)
}
