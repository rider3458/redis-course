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
	case "ZADD":
		return HandleZADD(cmd)
	case "ZSCORE":
		return HandleZSCORE(cmd)
	case "ZRANK":
		return HandleZRANK(cmd)
	case "ZRANGE":
		return HandleZRANGE(cmd)
	case "ZREM":
		return HandleZREM(cmd)
	case "CMS.INITBYDIM":
		return HandleCMSInitByDim(cmd)
	case "CMS.INITBYPROB":
		return HandleCMSInitByProb(cmd)
	case "CMS.INCRBY":
		return HandleCMSIncrBy(cmd)
	case "CMS.QUERY":
		return HandleCMSQuery(cmd)
	case "CMS.INFO":
		return HandleCMSInfo(cmd)
	case "CMS.MERGE":
		return HandleCMSMerge(cmd)
	case "BF.RESERVE":
		return HandleBFReserve(cmd)
	case "BF.ADD":
		return HandleBFAdd(cmd)
	case "BF.EXISTS":
		return HandleBFExists(cmd)
	case "BF.MADD":
		return HandleBFMAdd(cmd)
	case "BF.MEXISTS":
		return HandleBFMExists(cmd)
	case "BF.INFO":
		return HandleBFInfo(cmd)

	default:
		return EncodeUnknownCommand(cmd.Cmd)
	}
}
