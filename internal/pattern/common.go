package pattern

import (
	"cess-scheduler/internal/chain"
	apiv1 "cess-scheduler/internal/proof/apiv1"
	"sync/atomic"
)

const (
	Chan_Filler_len = 30
)

type Filler struct {
	FillerId string
	Path     string
	T        apiv1.FileTagT
	Sigmas   [][]byte `json:"sigmas"`
}

var Chan_Filler chan Filler
var Chan_FillerMeta chan chain.SpaceFileInfo
var TxStatus atomic.Value

func init() {
	Chan_Filler = make(chan Filler, Chan_Filler_len)
	Chan_FillerMeta = make(chan chain.SpaceFileInfo, 100)
}
