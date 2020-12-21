package contract

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// LogsBloom returns the bloom bytes for the given logs
func LogsBloom(logs []EvmLogs) []byte {
	var bin types.Bloom
	for _, log := range logs {
		bin.Add(log.Address.Bytes())
		for _, b := range log.Topics {
			bin.Add(b[:])
		}
	}
	return bin[:]
}
