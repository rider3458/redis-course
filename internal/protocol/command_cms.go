package protocol

import (
	"errors"
	"strconv"
	"strings"

	"github.com/rider3458/redis-course/internal/storage"
)

func HandleCMSInitByDim(cmd *Command) []byte {
	if len(cmd.Args) != 3 {
		return EncodeWrongArity("cms.initbydim")
	}
	width, err := positiveUint(cmd.Args[1])
	if err != nil {
		return EncodeError("ERR CMS: width must be a positive integer")
	}
	depth, err := positiveUint(cmd.Args[2])
	if err != nil {
		return EncodeError("ERR CMS: depth must be a positive integer")
	}
	return encodeCMSError(commandStore().CMSInit(cmd.Args[0], width, depth))
}

func HandleCMSInitByProb(cmd *Command) []byte {
	if len(cmd.Args) != 3 {
		return EncodeWrongArity("cms.initbyprob")
	}
	errorRate, err := strconv.ParseFloat(cmd.Args[1], 64)
	if err != nil || errorRate <= 0 || errorRate >= 1 {
		return EncodeError("ERR CMS: error rate must be between 0 and 1")
	}
	probability, err := strconv.ParseFloat(cmd.Args[2], 64)
	if err != nil || probability <= 0 || probability >= 1 {
		return EncodeError("ERR CMS: probability must be between 0 and 1")
	}
	return encodeCMSError(commandStore().CMSInitByProbability(cmd.Args[0], errorRate, probability))
}

func HandleCMSIncrBy(cmd *Command) []byte {
	if len(cmd.Args) < 3 || len(cmd.Args)%2 == 0 {
		return EncodeWrongArity("cms.incrby")
	}
	increments := make([]storage.CMSIncrement, 0, (len(cmd.Args)-1)/2)
	for index := 1; index < len(cmd.Args); index += 2 {
		value, err := positiveUint(cmd.Args[index+1])
		if err != nil {
			return EncodeError("ERR CMS: increment must be a positive integer")
		}
		increments = append(increments, storage.CMSIncrement{Item: cmd.Args[index], Value: value})
	}
	counts, err := commandStore().CMSIncrBy(cmd.Args[0], increments)
	if err != nil {
		return encodeCMSError(err)
	}
	return encodeCMSCounts(counts)
}

func HandleCMSQuery(cmd *Command) []byte {
	if len(cmd.Args) < 2 {
		return EncodeWrongArity("cms.query")
	}
	counts, err := commandStore().CMSQuery(cmd.Args[0], cmd.Args[1:])
	if err != nil {
		return encodeCMSError(err)
	}
	return encodeCMSCounts(counts)
}

func HandleCMSInfo(cmd *Command) []byte {
	if len(cmd.Args) != 1 {
		return EncodeWrongArity("cms.info")
	}
	info, err := commandStore().CMSInfo(cmd.Args[0])
	if err != nil {
		return encodeCMSError(err)
	}
	return EncodeArray([][]byte{
		EncodeBulkString("width"), EncodeInteger(int64(info.Width)),
		EncodeBulkString("depth"), EncodeInteger(int64(info.Depth)),
		EncodeBulkString("count"), EncodeInteger(int64(info.Count)),
	})
}

func HandleCMSMerge(cmd *Command) []byte {
	if len(cmd.Args) < 3 {
		return EncodeWrongArity("cms.merge")
	}
	numKeys, err := positiveUint(cmd.Args[1])
	if err != nil || numKeys > uint64(len(cmd.Args)-2) {
		return EncodeError("ERR CMS: invalid number of source keys")
	}
	sourceEnd := 2 + int(numKeys)
	sourceKeys := cmd.Args[2:sourceEnd]
	weights := make([]uint64, numKeys)
	for index := range weights {
		weights[index] = 1
	}
	if len(cmd.Args) != sourceEnd {
		if len(cmd.Args) != sourceEnd+1+int(numKeys) || strings.ToUpper(cmd.Args[sourceEnd]) != "WEIGHTS" {
			return EncodeError("ERR CMS: syntax error")
		}
		for index := range weights {
			weight, parseErr := positiveUint(cmd.Args[sourceEnd+1+index])
			if parseErr != nil {
				return EncodeError("ERR CMS: weight must be a positive integer")
			}
			weights[index] = weight
		}
	}
	return encodeCMSError(commandStore().CMSMerge(cmd.Args[0], sourceKeys, weights))
}

func positiveUint(value string) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("not a positive integer")
	}
	return parsed, nil
}

func encodeCMSCounts(counts []uint64) []byte {
	elements := make([][]byte, len(counts))
	for index, count := range counts {
		elements[index] = EncodeInteger(int64(count))
	}
	return EncodeArray(elements)
}

func encodeCMSError(err error) []byte {
	if err == nil {
		return EncodeSimpleString("OK")
	}
	switch err {
	case storage.ErrKeyExists:
		return EncodeError("ERR CMS: key already exists")
	case storage.ErrCMSKeyNotFound:
		return EncodeError("ERR CMS: key does not exist")
	case storage.ErrCMSIncompatible:
		return EncodeError("ERR CMS: sketches have different width/depth")
	case storage.ErrCMSOverflow:
		return EncodeError("ERR CMS: counter overflow")
	case storage.ErrWrongType:
		return EncodeWrongType()
	default:
		return EncodeError("ERR internal error")
	}
}
