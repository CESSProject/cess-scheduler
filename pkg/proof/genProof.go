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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Nik-U/pbc"
)

func (keyPair PBCKeyPair) GenProof(QSlice []QElement, t T, Phi []Sigma, Matrix [][]byte, SigRootHash []byte) <-chan GenProofResponse {
	responseCh := make(chan GenProofResponse, 1)
	var res GenProofResponse

	pairing, _ := pbc.NewPairingFromString(keyPair.SharedParams)
	g := pairing.NewG1().SetBytes(keyPair.SharedG)
	publicKey := pairing.NewG1().SetBytes(keyPair.Spk)

	tmp, err := json.Marshal(t.Tag)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}

	tagG1 := pairing.NewG1().SetFromStringHash(string(tmp), sha256.New())
	temp1 := pairing.NewGT().Pair(tagG1, publicKey)
	temp2 := pairing.NewGT().Pair(pairing.NewG1().SetBytes(t.SigAbove), g)
	if !temp1.Equals(temp2) {
		res.StatueMsg.StatusCode = ErrorParam
		res.StatueMsg.Msg = "Signature information verification error"
		responseCh <- res
		return responseCh
	} else {
		fmt.Println("Signature information verification success")
	}

	//Compute Mu
	mu := pairing.NewZr()
	sigma := pairing.NewG1()

	for i := 0; i < len(QSlice); i++ {
		//µ =Σ νi*mi ∈ Zp (i ∈ [1, n])
		mi := pairing.NewZr().SetBytes(Matrix[QSlice[i].I])
		vi := pairing.NewZr().SetBytes(QSlice[i].V)
		mu.Add(pairing.NewZr().Mul(mi, vi), mu)

		//σ =∏ σ^vi ∈ G (i ∈ [1, n])
		sigma_i := pairing.NewG1().SetBytes(Phi[QSlice[i].I])
		sigma.Mul(pairing.NewG1().PowZn(sigma_i, vi), sigma)

		hash_mi := pairing.NewG1().SetFromStringHash(string(Matrix[QSlice[i].I]), sha256.New())
		res.HashMi = append(res.HashMi, hash_mi.Bytes())
	}

	res.MU = mu.Bytes()
	res.Sigma = sigma.Bytes()
	res.SigRootHash = SigRootHash
	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "Success"
	responseCh <- res
	return responseCh
}

func SplitV2(fpath string, sep int64) (Data [][]byte, N int64, err error) {
	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, 0, err
	}
	file_size := int64(len(data))
	if sep > file_size {
		Data = append(Data, data)
		N = 1
		return
	}

	N = file_size / sep
	if file_size%sep != 0 {
		N += 1
	}

	for i := int64(0); i < N; i++ {
		if i != N-1 {
			Data = append(Data, data[i*sep:(i+1)*sep])
			continue
		}
		Data = append(Data, data[i*sep:])
		if l := sep - int64(len(data[i*sep:])); l > 0 {
			Data[i] = append(Data[i], make([]byte, l, l)...)
		}
	}

	return
}
