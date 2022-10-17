package pattern

import (
	"sync/atomic"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
)

type Filler struct {
	FillerId string
	Path     string
	T        pbc.FileTagT
	Sigmas   [][]byte `json:"sigmas"`
}

var ChainStatus atomic.Value
var C_Filler chan Filler
var C_FillerMeta chan chain.FillerMetaInfo

//var C_Max_Miner_Filler chan bool

func init() {
	C_Filler = make(chan Filler, configs.Num_Filler_Reserved)
	C_FillerMeta = make(chan chain.FillerMetaInfo, configs.Max_Filler_Meta)
	// C_Max_Miner_Filler = make(chan bool, configs.MAX_TCP_CONNECTION)
	// for i := 0; i < configs.MAX_TCP_CONNECTION; i++ {
	// 	C_Max_Miner_Filler <- true
	// }
}
