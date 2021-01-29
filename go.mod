module github.com/filecoin-project/specs-actors/v2

go 1.13

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/filecoin-project/go-address v0.0.5-0.20201103152444-f2023ef3f5bb
	github.com/filecoin-project/go-amt-ipld/v2 v2.1.1-0.20201006184820-924ee87a1349
	github.com/filecoin-project/go-bitfield v0.2.3-0.20201110211213-fe2c1862e816
	github.com/filecoin-project/go-hamt-ipld v0.1.5
	github.com/filecoin-project/go-hamt-ipld/v2 v2.0.0
	github.com/filecoin-project/go-state-types v0.0.0-20201102161440-c8033295a1fc
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filestar-project/evm-adapter v0.0.1
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs-util v0.0.2 // indirect
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-ipld-format v0.2.0 // indirect
	github.com/ipfs/go-log/v2 v2.1.2-0.20200626104915-0016c0b4b3e4
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.7 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/minio/sha256-simd v0.1.1
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-multihash v0.0.14
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	github.com/warpfork/go-wish v0.0.0-20200122115046-b9ea61034e4a // indirect
	github.com/whyrusleeping/cbor-gen v0.0.0-20200826160007-0b9f6c5fb163
	github.com/xorcare/golden v0.6.1-0.20191112154924-b87f686d7542
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/lint v0.0.0-20200130185559-910be7a94367 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/tools v0.0.0-20200827010519-17fd2f27a9e3 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/protobuf v1.25.0 // indirect
)

replace github.com/filestar-project/evm-adapter => gitlab.pixelplex.by/731-filecoin/evm-adapter v0.0.0-20210128144858-3bf9f9a3b523

replace github.com/filecoin-project/go-state-types => gitlab.pixelplex.by/731-filecoin/go-state-types v0.0.0-20201220030553-d54b189b0534
