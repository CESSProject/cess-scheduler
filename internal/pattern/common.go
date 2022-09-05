package pattern

import (
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
)

const (
	C_Filler_Maxlen = 30
)

type Filler struct {
	FillerId string
	Path     string
	T        pbc.FileTagT
	Sigmas   [][]byte `json:"sigmas"`
}

var C_Filler chan Filler
var C_FillerMeta chan chain.FillerMetaInfo

func init() {
	C_Filler = make(chan Filler, C_Filler_Maxlen)
	C_FillerMeta = make(chan chain.FillerMetaInfo, 100)
}
