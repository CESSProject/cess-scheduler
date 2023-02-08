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

import (
	"github.com/Nik-U/pbc"
)

func (keyPair PBCKeyPair) VerifyProof(t T, QSlice []QElement, m, sigma Sigma, mht MHTInfo) bool {
	pairing, _ := pbc.NewPairingFromString(keyPair.SharedParams)
	g := pairing.NewG1().SetBytes(keyPair.SharedG)
	v := pairing.NewG1().SetBytes(keyPair.Spk)

	multiply := pairing.NewG1()
	for i := 0; i < len(QSlice); i++ {
		hashMi := pairing.NewG1().SetBytes(mht.HashMi[i])
		// ∏ H(mi)^νi (i ∈ [1, n])
		multiply.Mul(multiply, pairing.NewG1().PowZn(hashMi, pairing.NewZr().SetBytes(QSlice[i].V)))
	}

	//u^µ
	u := pairing.NewG1().SetBytes(t.U)
	mu := pairing.NewZr().SetBytes(m)
	uPowMu := pairing.NewG1().PowZn(u, mu)

	left := pairing.NewGT()
	right := pairing.NewGT()
	left.Pair(pairing.NewG1().SetBytes(sigma), g)
	right.Pair(pairing.NewG1().Mul(multiply, uPowMu), v)

	return left.Equals(right)
}
