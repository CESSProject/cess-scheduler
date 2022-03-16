module scheduler-mining

go 1.16

require (
	github.com/CESSProject/cess-ffi v0.0.0-20220217052609-6c35c99d795c
	github.com/centrifuge/go-substrate-rpc-client v2.0.0+incompatible
	github.com/centrifuge/go-substrate-rpc-client/v3 v3.0.2
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.0
	github.com/deckarep/golang-set v1.7.1
	github.com/filecoin-project/go-fil-commcid v0.1.0 // indirect
	github.com/filecoin-project/go-state-types v0.1.1
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v5 v5.0.4 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/websocket v1.5.0
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/itering/scale.go v1.1.49
	github.com/klauspost/reedsolomon v1.9.14
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/tendermint/tendermint v0.35.2
	go.uber.org/zap v1.19.1
	google.golang.org/protobuf v1.27.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

replace github.com/CESSProject/cess-ffi => ./internal/ffi
