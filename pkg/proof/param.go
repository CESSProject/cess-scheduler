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

package proof

const (
	Success            = 200
	Error              = 201
	ErrorParam         = 202
	ErrorParamNotFound = 203
	ErrorInternal      = 204
)

//————————————————————————————————————————————————————————————————Implement KeyGen()————————————————————————————————————————————————————————————————

type PBCKeyPair struct {
	Spk          []byte
	Ssk          []byte
	SharedParams string
	SharedG      []byte
	ZrLength     uint
}

//————————————————————————————————————————————————————————————————Implement SigGen()————————————————————————————————————————————————————————————————

// Sigma is σ
type Sigma = []byte

// T be the file tag for F
type T struct {
	Tag
	SigAbove []byte
}

// Tag belongs to T
type Tag struct {
	Name []byte `json:"name"`
	N    int64  `json:"n"`
	U    []byte `json:"u"`
}

// SigGenResponse is result of SigGen() step
type SigGenResponse struct {
	T           T         `json:"t"`
	Phi         []Sigma   `json:"phi"`           //Φ = {σi}
	SigRootHash []byte    `json:"sig_root_hash"` //BLS
	StatueMsg   StatueMsg `json:"statue_msg"`
}

type StatueMsg struct {
	StatusCode int    `json:"status"`
	Msg        string `json:"msg"`
}

//————————————————————————————————————————————————————————————————Implement ChalGen()————————————————————————————————————————————————————————————————

type QElement struct {
	I int64  `json:"i"`
	V []byte `json:"v"`
}

//————————————————————————————————————————————————————————————————Implement GenProof()————————————————————————————————————————————————————————————————

type GenProofResponse struct {
	Sigma Sigma  `json:"sigmas"`
	MU    []byte `json:"mu"`
	MHTInfo
	SigRootHash []byte    `json:"sig_root_hash"`
	StatueMsg   StatueMsg `json:"statue_msg"`
}

type MHTInfo struct {
	HashMi [][]byte `json:"hash_mi"`
	Omega  []byte   `json:"omega"`
}
