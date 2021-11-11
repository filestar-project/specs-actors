package contract

import (
	"io"
	"io/ioutil"

	cbor "github.com/fxamacker/cbor/v2"
)

func marshal(w io.Writer, target interface{}) error {
	marsh, err := cbor.Marshal(target)
	if err != nil {
		return err
	}
	_, err = w.Write(marsh)
	return err
}

func unmarshal(r io.Reader, target interface{}) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return cbor.Unmarshal(data, target)
}

func (t *State) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &State{}
	}
	return marshal(w, target)
}

func (t *State) UnmarshalCBOR(r io.Reader) error {
	*t = State{}
	return unmarshal(r, t)
}

func (t *EvmLogs) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &EvmLogs{}
	}
	return marshal(w, target)
}

func (t *EvmLogs) UnmarshalCBOR(r io.Reader) error {
	*t = EvmLogs{}
	return unmarshal(r, t)
}

func (t *ContractParams) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &ContractParams{}
	}
	return marshal(w, target)
}

func (t *ContractParams) UnmarshalCBOR(r io.Reader) error {
	*t = ContractParams{}
	return unmarshal(r, t)
}

func (t *ContractResult) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &ContractResult{}
	}
	return marshal(w, target)
}

func (t *ContractResult) UnmarshalCBOR(r io.Reader) error {
	*t = ContractResult{}
	return unmarshal(r, t)
}

func (t *GetCodeResult) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &GetCodeResult{}
	}
	return marshal(w, target)
}

func (t *GetCodeResult) UnmarshalCBOR(r io.Reader) error {
	*t = GetCodeResult{}
	return unmarshal(r, t)
}

func (t *LogsEntry) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &LogsEntry{}
	}
	return marshal(w, target)
}

func (t *LogsEntry) UnmarshalCBOR(r io.Reader) error {
	*t = LogsEntry{}
	return unmarshal(r, t)
}

func (t *StorageInfo) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &StorageInfo{}
	}
	return marshal(w, target)
}

func (t *StorageInfo) UnmarshalCBOR(r io.Reader) error {
	*t = StorageInfo{}
	return unmarshal(r, t)
}

func (t *StorageResult) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &StorageResult{}
	}
	return marshal(w, target)
}

func (t *StorageResult) UnmarshalCBOR(r io.Reader) error {
	*t = StorageResult{}
	return unmarshal(r, t)
}

func (t *TipsetState) MarshalCBOR(w io.Writer) error {
	target := t
	if target == nil {
		target = &TipsetState{}
	}
	return marshal(w, target)
}

func (t *TipsetState) UnmarshalCBOR(r io.Reader) error {
	*t = TipsetState{}
	return unmarshal(r, t)
}
