/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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
