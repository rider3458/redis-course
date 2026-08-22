package protocol

// HandlePING processes the PING command.
//
// TODO: Implement command behavior.
func HandlePING(cmd *Command) []byte {
	argsLen := len(cmd.Args)

	switch argsLen {
	case 0:
		return EncodeSimpleString("PONG")
	case 1:
		return EncodeSimpleString(cmd.Args[0])
	default:
		return EncodeWrongArity("ping")
	}
}
