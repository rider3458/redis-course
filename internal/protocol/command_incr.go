package protocol

// HandleINCR processes the INCR command.
func HandleINCR(cmd *Command) []byte {
	store := commandStore()
	if len(cmd.Args) != 1 {
		return EncodeWrongArity("incr")
	}

	next, err := store.Incr(cmd.Args[0])
	if err != nil {
		return EncodeNotInteger()
	}

	return EncodeInteger(next)
}
