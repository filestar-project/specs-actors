// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package verifreg

import (
	"fmt"
	"io"
	"sort"

	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf
var _ = cid.Undef
var _ = sort.Sort

var lengthBufState = []byte{131}

func (t *State) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write(lengthBufState); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.RootKey (address.Address) (struct)
	if err := t.RootKey.MarshalCBOR(w); err != nil {
		return err
	}

	// t.Verifiers (cid.Cid) (struct)

	if err := cbg.WriteCidBuf(scratch, w, t.Verifiers); err != nil {
		return xerrors.Errorf("failed to write cid field t.Verifiers: %w", err)
	}

	// t.VerifiedClients (cid.Cid) (struct)

	if err := cbg.WriteCidBuf(scratch, w, t.VerifiedClients); err != nil {
		return xerrors.Errorf("failed to write cid field t.VerifiedClients: %w", err)
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

	if extra != 3 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.RootKey (address.Address) (struct)

	{

		if err := t.RootKey.UnmarshalCBOR(br); err != nil {
			return xerrors.Errorf("unmarshaling t.RootKey: %w", err)
		}

	}
	// t.Verifiers (cid.Cid) (struct)

	{

		c, err := cbg.ReadCid(br)
		if err != nil {
			return xerrors.Errorf("failed to read cid field t.Verifiers: %w", err)
		}

		t.Verifiers = c

	}
	// t.VerifiedClients (cid.Cid) (struct)

	{

		c, err := cbg.ReadCid(br)
		if err != nil {
			return xerrors.Errorf("failed to read cid field t.VerifiedClients: %w", err)
		}

		t.VerifiedClients = c

	}
	return nil
}
