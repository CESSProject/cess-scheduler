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

package node

import (
	"fmt"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// task_ValidateProof is used to verify the proof data
func (node *Node) task_ValidateProof(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			node.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	var (
		err    error
		txhash string
		//poDR2verify   pbc.PoDR2Verify
		proofs        = make([]chain.Proof, 0)
		verifyResults = make([]chain.ProofResult, 0)
	)
	node.Logs.Verify("info", errors.New(">>> Start task_ValidateProof <<<"))

	for {
		for node.Chain.GetChainStatus() {
			// Get proofs from chain
			proofs, err = node.Chain.GetProofs()
			if err != nil {
				if err.Error() != chain.ERR_RPC_EMPTY_VALUE.Error() {
					node.Logs.Verify("error", err)
				}
			}
			if len(proofs) == 0 {
				time.Sleep(time.Minute)
				continue
			}

			node.Logs.Verify("info", fmt.Errorf("There are %d proofs", len(proofs)))

			for i := 0; i < len(proofs); i++ {
				// Proof results reach the maximum number of submissions
				if len(verifyResults) >= configs.Max_SubProofResults {
					// submit proof results
					for {
						txhash, _ = node.Chain.SubmitProofResults(verifyResults)
						if txhash != "" {
							node.Logs.Verify("info", fmt.Errorf("Proof result submitted: %v", txhash))
							break
						}
						time.Sleep(configs.BlockInterval)
					}
					verifyResults = make([]chain.ProofResult, 0)
				}

				// Organizational random number structure
				qSlice, err := pbc.PoDR2ChallengeGenerateFromChain(
					proofs[i].Challenge_info.Block_list,
					proofs[i].Challenge_info.Random,
				)
				if err != nil {
					node.Logs.Verify("error", fmt.Errorf("qslice: %v", err))
				}

				var t pbc.T
				var mht pbc.MHTInfo
				mht.HashMi = make([][]byte, len(proofs[i].HashMi))
				for j := 0; j < len(proofs[i].HashMi); j++ {
					mht.HashMi[j] = make([]byte, 0)
					mht.HashMi[j] = append(mht.HashMi[j], proofs[i].HashMi[j]...)
				}
				t.U = proofs[i].U
				// validate proof
				result := pbc.PbcKey.VerifyProof(t, qSlice, proofs[i].Mu, proofs[i].Sigma, mht)
				resultTemp := chain.ProofResult{}
				resultTemp.PublicKey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				resultTemp.Shard_id = proofs[i].Challenge_info.Shard_id
				resultTemp.Result = types.Bool(result)
				verifyResults = append(verifyResults, resultTemp)
			}

			// submit proof results// submit proof results
			if len(verifyResults) > 0 {
				for {
					txhash, _ = node.Chain.SubmitProofResults(verifyResults)
					if txhash != "" {
						node.Logs.Verify("info", fmt.Errorf("Proof result submitted: %v", txhash))
						break
					}
					time.Sleep(configs.BlockInterval)
				}
				verifyResults = make([]chain.ProofResult, 0)
			}
			time.Sleep(time.Second * configs.BlockInterval)
		}
		time.Sleep(configs.BlockInterval)
	}
}
