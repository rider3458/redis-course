package protocol

import (
	"strconv"

	"github.com/rider3458/redis-course/internal/storage"
)

func HandleBFReserve(cmd *Command) []byte {
	if len(cmd.Args) != 3 {
		return EncodeWrongArity("bf.reserve")
	}
	errorRate, err := strconv.ParseFloat(cmd.Args[1], 64)
	if err != nil || errorRate <= 0 || errorRate >= 1 {
		return EncodeError("ERR BF: error rate must be between 0 and 1")
	}
	capacity, err := positiveUint(cmd.Args[2])
	if err != nil {
		return EncodeError("ERR BF: capacity must be a positive integer")
	}
	return encodeBloomFilterError(commandStore().BFReserve(cmd.Args[0], errorRate, capacity))
}

func HandleBFAdd(cmd *Command) []byte {
	if len(cmd.Args) != 2 {
		return EncodeWrongArity("bf.add")
	}
	added, err := commandStore().BFAdd(cmd.Args[0], cmd.Args[1])
	if err != nil {
		return encodeBloomFilterError(err)
	}
	return EncodeInteger(boolToInt64(added))
}

func HandleBFExists(cmd *Command) []byte {
	if len(cmd.Args) != 2 {
		return EncodeWrongArity("bf.exists")
	}
	exists, err := commandStore().BFExists(cmd.Args[0], cmd.Args[1])
	if err != nil {
		return encodeBloomFilterError(err)
	}
	return EncodeInteger(boolToInt64(exists))
}

func HandleBFMAdd(cmd *Command) []byte {
	if len(cmd.Args) < 2 {
		return EncodeWrongArity("bf.madd")
	}
	added, err := commandStore().BFMultiAdd(cmd.Args[0], cmd.Args[1:])
	if err != nil {
		return encodeBloomFilterError(err)
	}
	return encodeBloomFilterBooleans(added)
}

func HandleBFMExists(cmd *Command) []byte {
	if len(cmd.Args) < 2 {
		return EncodeWrongArity("bf.mexists")
	}
	exists, err := commandStore().BFMultiExists(cmd.Args[0], cmd.Args[1:])
	if err != nil {
		return encodeBloomFilterError(err)
	}
	return encodeBloomFilterBooleans(exists)
}

func HandleBFInfo(cmd *Command) []byte {
	if len(cmd.Args) != 1 {
		return EncodeWrongArity("bf.info")
	}
	info, err := commandStore().BFInfo(cmd.Args[0])
	if err != nil {
		return encodeBloomFilterError(err)
	}
	return EncodeArray([][]byte{
		EncodeBulkString("Capacity"), EncodeInteger(int64(info.Capacity)),
		EncodeBulkString("Size"), EncodeInteger(int64(info.Size)),
		EncodeBulkString("Number of filters"), EncodeInteger(1),
		EncodeBulkString("Number of items inserted"), EncodeInteger(int64(info.Count)),
		EncodeBulkString("Expansion rate"), EncodeInteger(0),
	})
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func encodeBloomFilterBooleans(values []bool) []byte {
	elements := make([][]byte, len(values))
	for index, value := range values {
		elements[index] = EncodeInteger(boolToInt64(value))
	}
	return EncodeArray(elements)
}

func encodeBloomFilterError(err error) []byte {
	if err == nil {
		return EncodeSimpleString("OK")
	}
	switch err {
	case storage.ErrKeyExists:
		return EncodeError("ERR BF: key already exists")
	case storage.ErrBloomFilterKeyNotFound:
		return EncodeError("ERR BF: key does not exist")
	case storage.ErrBloomFilterIncompatible:
		return EncodeError("ERR BF: invalid filter configuration")
	case storage.ErrWrongType:
		return EncodeWrongType()
	default:
		return EncodeError("ERR internal error")
	}
}
