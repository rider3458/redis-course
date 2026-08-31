package protocol

// EncodeUnknownCommand returns an encoded unknown-command response.
func EncodeUnknownCommand(commandName string) []byte {
	return EncodeError("ERR unknown command '" + commandName + "'")
}

// EncodeWrongArity returns an encoded wrong-arity response.
func EncodeWrongArity(commandName string) []byte {
	return EncodeError("ERR wrong number of arguments for '" + commandName + "' command")
}

// EncodeNotInteger returns an encoded value-type response for integer commands.
func EncodeNotInteger() []byte {
	return EncodeError("ERR value is not an integer or out of range")
}

// EncodeWrongType returns an encoded wrong-type error response.
func EncodeWrongType() []byte {
	return EncodeError("WRONGTYPE Operation against a key holding the wrong kind of value")
}
