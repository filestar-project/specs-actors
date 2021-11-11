package contract

import (
	"github.com/ethereum/go-ethereum/core/types"
)

type logsCollector struct {
	logs []EvmLogs
}

type LogsEntry struct {
	Empty     bool
	Height    int64
	Logs      []EvmLogs
	LogsBloom types.Bloom
}

var collector logsCollector

func appendLogs(log []EvmLogs) {
	collector.logs = append(collector.logs, log...)
}

func flushLogs() *LogsEntry {
	defer clearCollector()
	if collector.logs != nil {
		return &LogsEntry{Empty: false, Logs: collector.logs, LogsBloom: types.BytesToBloom(LogsBloom(collector.logs))}
	}
	return &LogsEntry{Empty: true, Logs: []EvmLogs{}, LogsBloom: types.BytesToBloom(LogsBloom([]EvmLogs{}))}
}

func clearCollector() {
	collector.logs = nil
}
