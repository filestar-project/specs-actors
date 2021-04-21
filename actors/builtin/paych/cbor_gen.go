// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package paych

import (
	"fmt"
	"io"
	"sort"

	abi "github.com/filecoin-project/go-state-types/abi"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf
var _ = cid.Undef
var _ = sort.Sort

var lengthBufState = []byte{134}

func (t *State) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBufState); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.From (address.Address) (struct)
	if err := t.From.MarshalCBOR(w); err != nil {
		return err
	}

	// t.To (address.Address) (struct)
	if err := t.To.MarshalCBOR(w); err != nil {
		return err
	}

	// t.ToSend (big.Int) (struct)
	if err := t.ToSend.MarshalCBOR(w); err != nil {
		return err
	}

	// t.SettlingAt (abi.ChainEpoch) (int64)
	if t.SettlingAt >= 0 {
		if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajUnsignedInt, uint64(t.SettlingAt)); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajNegativeInt, uint64(-t.SettlingAt-1)); err != nil {
			return err
		}
	}

	// t.MinSettleHeight (abi.ChainEpoch) (int64)
	if t.MinSettleHeight >= 0 {
		if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajUnsignedInt, uint64(t.MinSettleHeight)); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajNegativeInt, uint64(-t.MinSettleHeight-1)); err != nil {
			return err
		}
	}

	// t.LaneStates (cid.Cid) (struct)

	if err := cbg.WriteCidBuf(scratch, w, t.LaneStates); err != nil {
		return xerrors.Errorf("failed to write cid field t.LaneStates: %w", err)
	}

	return nil
}

func (t *State) UnmarshalCBOR(r io.Reader) error {
	*t = State{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 6 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.From (address.Address) (struct)

	{

		if err := t.From.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.From: %w", err)
		}

	}
	// t.To (address.Address) (struct)

	{

		if err := t.To.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.To: %w", err)
		}

	}
	// t.ToSend (big.Int) (struct)

	{

		if err := t.ToSend.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.ToSend: %w", err)
		}

	}
	// t.SettlingAt (abi.ChainEpoch) (int64)
	{
		maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
		var extraI int64
		if err != nil {
			return err
		}
		switch maj {
		case cbg.MajUnsignedInt:
			extraI = int64(extra)
			if extraI < 0 {
				return fmt.Errorf("int64 positive overflow")
			}
		case cbg.MajNegativeInt:
			extraI = int64(extra)
			if extraI < 0 {
				return fmt.Errorf("int64 negative oveflow")
			}
			extraI = -1 - extraI
		default:
			return fmt.Errorf("wrong type for int64 field: %d", maj)
		}

		t.SettlingAt = abi.ChainEpoch(extraI)
	}
	// t.MinSettleHeight (abi.ChainEpoch) (int64)
	{
		maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
		var extraI int64
		if err != nil {
			return err
		}
		switch maj {
		case cbg.MajUnsignedInt:
			extraI = int64(extra)
			if extraI < 0 {
				return fmt.Errorf("int64 positive overflow")
			}
		case cbg.MajNegativeInt:
			extraI = int64(extra)
			if extraI < 0 {
				return fmt.Errorf("int64 negative oveflow")
			}
			extraI = -1 - extraI
		default:
			return fmt.Errorf("wrong type for int64 field: %d", maj)
		}

		t.MinSettleHeight = abi.ChainEpoch(extraI)
	}
	// t.LaneStates (cid.Cid) (struct)

	{

		c, err := cbg.ReadCid(br)
		if err != nil {
			return xerrors.Errorf("failed to read cid field t.LaneStates: %w", err)
		}

		t.LaneStates = c

	}
	return nil
}

var lengthBufLaneState = []byte{130}

func (t *LaneState) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBufLaneState); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.Redeemed (big.Int) (struct)
	if err := t.Redeemed.MarshalCBOR(w); err != nil {
		return err
	}

	// t.Nonce (uint64) (uint64)

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajUnsignedInt, uint64(t.Nonce)); err != nil {
		return err
	}

	return nil
}

func (t *LaneState) UnmarshalCBOR(r io.Reader) error {
	*t = LaneState{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 2 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Redeemed (big.Int) (struct)

	{

		if err := t.Redeemed.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.Redeemed: %w", err)
		}

	}
	// t.Nonce (uint64) (uint64)

	{

		maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
		if err != nil {
			return err
		}
		if maj != cbg.MajUnsignedInt {
			return fmt.Errorf("wrong type for uint64 field")
		}
		t.Nonce = uint64(extra)

	}
	return nil
}

var lengthBufUpdateChannelStateParams = []byte{130}

func (t *UpdateChannelStateParams) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBufUpdateChannelStateParams); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.Sv (paych.SignedVoucher) (struct)
	if err := t.Sv.MarshalCBOR(w); err != nil {
		return err
	}

	// t.Secret ([]uint8) (slice)
	if len(t.Secret) > cbg.ByteArrayMaxLen {
		return xerrors.Errorf("Byte array in field t.Secret was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajByteString, uint64(len(t.Secret))); err != nil {
		return err
	}

	if _, err := w.Write(t.Secret[:]); err != nil {
		return err
	}
	return nil
}

func (t *UpdateChannelStateParams) UnmarshalCBOR(r io.Reader) error {
	*t = UpdateChannelStateParams{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 2 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Sv (paych.SignedVoucher) (struct)

	{

		if err := t.Sv.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.Sv: %w", err)
		}

	}
	// t.Secret ([]uint8) (slice)

	maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}

	if extra > cbg.ByteArrayMaxLen {
		return fmt.Errorf("t.Secret: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}

	if extra > 0 {
		t.Secret = make([]uint8, extra)
	}

	if _, err := io.ReadFull(br, t.Secret[:]); err != nil {
		return err
	}
	return nil
}
