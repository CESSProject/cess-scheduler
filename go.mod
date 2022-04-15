module cess-scheduler

go 1.16

require (
	github.com/BurntSushi/toml v1.0.0 // indirect
	github.com/CESSProject/cess-ffi v0.0.0-20220223120821-e42303f4cccf
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.0
	github.com/deckarep/golang-set v1.7.1
	github.com/filecoin-project/go-fil-commcid v0.1.0 // indirect
	github.com/filecoin-project/go-state-types v0.1.1
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v5 v5.0.4 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/gorilla/websocket v1.5.0
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/klauspost/reedsolomon v1.9.14
	github.com/kr/pretty v0.3.0 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/onsi/gomega v1.17.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/rs/cors v1.8.2 // indirect
	github.com/spf13/afero v1.8.0 // indirect
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce
	golang.org/x/net v0.0.0-20211208012354-db4efeb81f4b // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/tools v0.1.9 // indirect
	google.golang.org/protobuf v1.27.1
	gopkg.in/ini.v1 v1.66.3 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

replace github.com/CESSProject/cess-ffi => ./internal/ffi
