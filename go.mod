module github.com/filecoin-project/specs-actors/v3

go 1.13

require (
	github.com/ethereum/go-ethereum v1.9.25 // indirect
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-amt-ipld/v2 v2.1.1-0.20201006184820-924ee87a1349
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-hamt-ipld v0.1.5
	github.com/filecoin-project/go-hamt-ipld/v2 v2.0.0
	github.com/filecoin-project/go-state-types v0.1.0
	github.com/filecoin-project/lotus v1.5.0
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filestar-project/evm-adapter v0.0.1
	github.com/fxamacker/cbor/v2 v2.3.0
	github.com/hashicorp/go-memdb v1.3.2
	github.com/holiman/uint256 v1.1.1
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-log/v2 v2.1.2-0.20200626104915-0016c0b4b3e4
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/minio/sha256-simd v0.1.1
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/multiformats/go-multihash v0.0.14
	github.com/objectbox/objectbox-go v1.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca // indirect
	github.com/whyrusleeping/cbor-gen v0.0.0-20210219115102-f37d292932f2
	github.com/xorcare/golden v0.6.1-0.20191112154924-b87f686d7542
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/specs-actors/v2 => ./

replace github.com/ethereum/go-ethereum => github.com/filestar-project/geth v0.0.0-20210421091648-739792a01e4a

replace github.com/filestar-project/evm-adapter => github.com/filestar-project/evm_adapter v0.0.0-20210421144238-f86b43754ed3

replace github.com/filecoin-project/go-state-types => github.com/filestar-project/go-state-types v0.0.0-20201220030553-d54b189b0534
