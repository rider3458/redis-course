package protocol

import (
	"math"
	"strconv"

	"github.com/rider3458/redis-course/internal/storage"
)

// HandleZADD processes the ZADD command.
func HandleZADD(cmd *Command) []byte {
	if len(cmd.Args) < 3 || len(cmd.Args)%2 == 0 {
		return EncodeWrongArity("zadd")
	}

	members := make(map[string]float64, (len(cmd.Args)-1)/2)
	for index := 1; index < len(cmd.Args); index += 2 {
		score, err := strconv.ParseFloat(cmd.Args[index], 64)
		if err != nil || math.IsNaN(score) {
			return EncodeError("ERR value is not a valid float")
		}
		members[cmd.Args[index+1]] = score
	}

	added, err := commandStore().ZAdd(cmd.Args[0], members)
	if err != nil {
		if err == storage.ErrWrongType {
			return EncodeWrongType()
		}
		return EncodeError("ERR internal error")
	}
	return EncodeInteger(int64(added))
}
