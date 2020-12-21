package contract

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/mitchellh/go-homedir"
	leveldb "github.com/syndtr/goleveldb/leveldb"
)

type LogsStoreManager struct {
	db          *leveldb.DB
	LogsEntries chan interface{}
}

var logsStoreManager *LogsStoreManager

func GetLogsManager() (*LogsStoreManager, error) {
	var err error
	if logsStoreManager == nil {
		logsStoreManager, err = newLogsManager()
	}
	return logsStoreManager, err
}

func newLogsManager() (*LogsStoreManager, error) {
	err := setRandomPath()
	if err != nil {
		return nil, err
	}
	dir, err := homedir.Expand(PathToRepo)
	if err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(dir+"/LogsDB", nil)
	return &LogsStoreManager{db: db, LogsEntries: nil}, err
}

func (manager *LogsStoreManager) SetLogsChannel(ch chan interface{}) {
	manager.LogsEntries = ch
}

func (manager *LogsStoreManager) UpdateHeightLogs(height int64) error {
	encodedLogs := new(bytes.Buffer)
	logEntry := flushLogs()
	logEntry.Height = height
	if manager.LogsEntries != nil && !logEntry.Empty {
		manager.LogsEntries <- *logEntry
	}
	if err := logEntry.MarshalCBOR(encodedLogs); err != nil {
		return err
	}
	return manager.db.Put([]byte(strconv.FormatInt(height, 16)), encodedLogs.Bytes(), nil)
}

func (manager *LogsStoreManager) GetHeightLogs(height int64) (LogsEntry, error) {
	value, err := manager.db.Get([]byte(strconv.FormatInt(height, 16)), nil)
	if err != nil {
		return LogsEntry{}, err
	}
	var result LogsEntry
	if err := result.UnmarshalCBOR(bytes.NewReader(value)); err != nil {
		return LogsEntry{}, fmt.Errorf("can't unmarshal result from value in database: %w", err)
	}
	return result, nil
}

func CloseLogsManager() error {
	if logsStoreManager != nil {
		return logsStoreManager.db.Close()
	}
	return nil
}
