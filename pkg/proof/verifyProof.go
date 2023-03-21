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
	"fmt"
	"math/big"
	"sync"
)

func (keyPair RSAKeyPair) VerifyProof(t T, QSlice []QElement, mu, sigma Sigma, mht MHTInfo, sigRootHash []byte) bool {

	//var mi []merkletree.NodeSerializable
	//var auxiliary []merkletree.NodeSerializable
	multiply := new(big.Int).SetInt64(1)
	var lock sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < len(QSlice); i++ {
		wg.Add(1)
		go func(i int) {
			hashMi := new(big.Int).SetBytes(mht.HashMi[i])
			// ∏ H(mi)^νi (i ∈ [1, n])
			pow := new(big.Int).Exp(hashMi, new(big.Int).SetBytes(QSlice[i].V), keyPair.Spk.N)
			lock.Lock()
			multiply.Mul(multiply, pow)
			lock.Unlock()
			wg.Done()
		}(i)

		////for verify MHT root
		//var n merkletree.NodeSerializable
		//n.Hash = MHT.HashMi[i]
		//n.Index = QSlice[i].I
		//n.Height = 0
		//mi = append(mi, n)
	}
	wg.Wait()

	//err := json.Unmarshal(MHT.Omega, &auxiliary)
	//if err != nil {
	//	panic(err)
	//}

	//proofNode := append(mi, auxiliary...)
	//for _,v:=range proofNode{
	//	fmt.Println(hex.EncodeToString(v.Hash))
	//}
	//
	//root, err := merkletree.NewTreeWithAuxiliaryNode(merkletree.RebuildNodeList(&proofNode), sha256.New)
	//if err != nil {
	//	panic(err)
	//}
	////verify hash root signature
	//if !bytes.Equal(root.Hash, SigRootHash) {
	//	fmt.Println("root signature verify fail")
	//	return false
	//}
	//u^µ
	// u := new(big.Int).SetBytes(t.U)
	// mus := new(big.Int).SetBytes(mu)
	// uPowMu := new(big.Int).Exp(u, mus, keyPair.Spk.N)

	// return new(big.Int).Mod(new(big.Int).Mul(multiply, uPowMu), keyPair.Spk.N).Cmp(new(big.Int).Exp(new(big.Int).SetBytes(sigma), new(big.Int).SetInt64(int64(keyPair.Spk.E)), keyPair.Spk.N)) == 0
	u := new(big.Int).SetBytes(t.U)
	mus := new(big.Int).SetBytes(mu)
	uPowMu := new(big.Int).Exp(u, mus, keyPair.Spk.N)

	fmt.Println("key.Spk.N:", keyPair.Spk.N)
	fmt.Println("key.Spk.E:", keyPair.Spk.E)
	fmt.Println("T.tag.u:", t.U)
	fmt.Println("T.tag.n:", t.N)
	fmt.Println("T.tag.name:", t.Name)
	fmt.Println("T.SigAbove :", t.SigAbove)
	fmt.Println("T.sigRootHash :", sigRootHash)
	fmt.Println("random:", QSlice[0])
	return new(big.Int).Mod(new(big.Int).Mul(multiply, uPowMu), keyPair.Spk.N).Cmp(new(big.Int).Exp(new(big.Int).SetBytes(sigma), new(big.Int).SetInt64(int64(keyPair.Spk.E)), keyPair.Spk.N)) == 0
}
