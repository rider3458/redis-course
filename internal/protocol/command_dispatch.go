package protocol

// DispatchCommand routes a parsed command to a command handler.
func DispatchCommand(cmd *Command) []byte {
	if cmd == nil {
		return EncodeError("ERR internal error")
	}

	switch cmd.Cmd {
	case "PING":
		return HandlePING(cmd)
	case "SET":
		return HandleSET(cmd)
	case "GET":
		return HandleGET(cmd)
	case "EXPIRE":
		return HandleEXPIRE(cmd)
	case "TTL":
		return HandleTTL(cmd)
	case "DEL":
		return HandleDEL(cmd)
	case "EXISTS":
		return HandleEXISTS(cmd)
	case "INCR":
		return HandleINCR(cmd)
	case "SADD":
		return HandleSADD(cmd)
	case "SREM":
		return HandleSREM(cmd)
	case "SISMEMBER":
		return HandleSISMEMBER(cmd)
	case "SMEMBERS":
		return HandleSMEMBERS(cmd)

	default:
		return EncodeUnknownCommand(cmd.Cmd)
	}
}
