package contract

import (
	"bytes"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	leveldb "github.com/syndtr/goleveldb/leveldb"
)

type TipsetState struct {
	State    cid.Cid
	Rectroot cid.Cid
}

type TipSetStateEntity struct {
	Height uint64
	State  TipsetState
}

type TipSetStateManager struct {
	db *leveldb.DB
}

var tipSetStateManager *TipSetStateManager

func GetTipSetStateManager() (*TipSetStateManager, error) {
	var err error
	if tipSetStateManager == nil {
		tipSetStateManager, err = newTipSetStateManager()
	}
	return tipSetStateManager, err
}

func newTipSetStateManager() (*TipSetStateManager, error) {
	err := setRandomPath()
	if err != nil {
		return nil, err
	}
	dir, err := homedir.Expand(PathToRepo)
	if err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(dir+"/TipSetStateDB", nil)
	return &TipSetStateManager{db: db}, err
}

func (manager *TipSetStateManager) UpdateState(height string, state TipsetState) error {
	buf := new(bytes.Buffer)
	if err := state.MarshalCBOR(buf); err != nil {
		return fmt.Errorf("failed to cbor marshal state: %w", err)
	}
	return manager.db.Put([]byte(height), buf.Bytes(), nil)
}

func (manager *TipSetStateManager) GetState(height string) (TipsetState, error) {
	data, err := manager.db.Get([]byte(height), nil)
	if err != nil {
		return TipsetState{}, err
	}
	var state TipsetState
	err = state.UnmarshalCBOR(bytes.NewReader(data))
	return state, err
}

func CloseTipSetStateManager() error {
	if tipSetStateManager != nil {
		return tipSetStateManager.db.Close()
	}
	return nil
}
