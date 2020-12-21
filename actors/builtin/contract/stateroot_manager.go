package contract

import (
	"github.com/mitchellh/go-homedir"
	leveldb "github.com/syndtr/goleveldb/leveldb"
)

type StateRoot struct {
	Height uint64
	Root   []byte
}

type StateRootManager struct {
	db *leveldb.DB
}

var stateRootManager *StateRootManager

func GetStateRootManager() (*StateRootManager, error) {
	var err error
	if stateRootManager == nil {
		stateRootManager, err = newStateRootManager()
	}
	return stateRootManager, err
}

func newStateRootManager() (*StateRootManager, error) {
	err := setRandomPath()
	if err != nil {
		return nil, err
	}
	dir, err := homedir.Expand(PathToRepo)
	if err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(dir+"/StateRootDB", nil)
	return &StateRootManager{db: db}, err
}

func (manager *StateRootManager) UpdateRoot(height string) error {
	return manager.db.Put([]byte(height), getCurrentStateRoot().Bytes(), nil)
}

func (manager *StateRootManager) GetRoot(height string) ([]byte, error) {
	return manager.db.Get([]byte(height), nil)
}

func CloseManager() error {
	if stateRootManager != nil {
		return stateRootManager.db.Close()
	}
	return nil
}
