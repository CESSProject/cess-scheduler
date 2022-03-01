module scheduler-mining

go 1.16

require (
	github.com/CESSProject/cess-ffi v0.0.0-20220217052609-6c35c99d795c
	github.com/HaoyuHu/gosimhash v0.0.0-20171130162857-733e771035c5
	github.com/centrifuge/go-substrate-rpc-client/v3 v3.0.2
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.0
	github.com/corona10/goimagehash v1.0.3
	github.com/dchest/siphash v1.2.2 // indirect
	github.com/filecoin-project/go-fil-commcid v0.1.0 // indirect
	github.com/filecoin-project/go-state-types v0.1.1
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v5 v5.0.4 // indirect
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/klauspost/reedsolomon v1.9.14
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.0
	github.com/yanyiwu/gojieba v1.1.2 // indirect
	go.uber.org/zap v1.19.1
	golang.org/x/image v0.0.0-20190802002840-cff245a6509b
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

replace github.com/CESSProject/cess-ffi => ./internal/ffi

