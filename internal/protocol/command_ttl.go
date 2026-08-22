package protocol

// HandleTTL processes the TTL command.
func HandleTTL(cmd *Command) []byte {
	if len(cmd.Args) != 1 {
		return EncodeWrongArity("ttl")
	}

	ttl := commandStore().TTL(cmd.Args[0])
	return EncodeInteger(ttl)
}
