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
	"github.com/CESSProject/cess-scheduler/pkg/proof"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// task_ValidateProof is used to verify the proof data
func (n *Node) task_ValidateProof(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()

	var (
		err           error
		proofs        []chain.Proof
		key           *proof.RSAKeyPair
		verifyResults = make([]chain.ProofResult, 0)
	)

	n.Logs.Verify("info", errors.New(">>> Start task_ValidateProof <<<"))

	for key == nil {
		key, _ = proof.GetKey(n.Cache)
		time.Sleep(time.Second)
	}

	for {
		// if n.Chain.GetChainStatus() {
		// 	time.Sleep(time.Minute)
		// 	continue
		// }
		// Get proofs from chain
		proofs, err = n.Chain.GetProofs()
		if err != nil {
			if err.Error() != chain.ERR_RPC_EMPTY_VALUE.Error() {
				n.Logs.Verify("err", err)
				time.Sleep(configs.BlockInterval)
			} else {
				time.Sleep(time.Minute)
			}
			continue
		}

		n.Logs.Verify("info", fmt.Errorf("There are %d proofs", len(proofs)))

		for i := 0; i < len(proofs); i++ {
			// Proof results reach the maximum number of submissions
			if len(verifyResults) >= configs.Max_SubProofResults {
				n.Logs.Verify("info", fmt.Errorf("Submit %d proofs", len(verifyResults)))
				n.submitProofResult(verifyResults)
				verifyResults = make([]chain.ProofResult, 0)
			}
			res := n.verifyProof(key, proofs[i])
			verifyResults = append(verifyResults, res)
		}

		// submit proof results
		n.submitProofResult(verifyResults)
		verifyResults = make([]chain.ProofResult, 0)
		time.Sleep(configs.BlockInterval)
	}
}

func (n *Node) verifyProof(key *proof.RSAKeyPair, prf chain.Proof) chain.ProofResult {
	var (
		err    error
		qSlice []proof.QElement
		result chain.ProofResult
		t      proof.T
		mht    proof.MHTInfo
	)

	result.PublicKey = prf.Miner_pubkey
	result.FileId = prf.Challenge_info.File_id
	result.Shard_id = prf.Challenge_info.Shard_id
	if len(prf.U) == 0 || len(prf.Sigma) == 0 || len(prf.Omega) == 0 {
		result.Result = types.Bool(false)
	} else {
		// Organizational random number structure
		qSlice, err = proof.PoDR2ChallengeGenerateFromChain(
			prf.Challenge_info.Block_list,
			prf.Challenge_info.Random,
		)
		if err != nil {
			n.Logs.Verify("err", fmt.Errorf("qslice: %v", err))
			result.Result = types.Bool(true)
			return result
		}

		t.U = prf.U
		mht.HashMi = make([][]byte, len(prf.HashMi))
		for j := 0; j < len(prf.HashMi); j++ {
			mht.HashMi[j] = make([]byte, 0)
			mht.HashMi[j] = append(mht.HashMi[j], prf.HashMi[j]...)
		}
		mht.Omega = prf.Omega
		result.Result = types.Bool(key.VerifyProof(t, qSlice, prf.Mu, prf.Sigma, mht, prf.SigRootHash))
	}
	// validate proof
	return result
}

func (n *Node) submitProofResult(proofs []chain.ProofResult) {
	var (
		err      error
		tryCount uint8
		txhash   string
	)
	// submit proof results
	if len(proofs) > 0 {
		for {
			txhash, err = n.Chain.SubmitProofResults(proofs)
			if err != nil {
				tryCount++
				n.Logs.Verify("err", fmt.Errorf("Proof result submitted err: %v", err))
			}
			if txhash != "" {
				n.Logs.Verify("info", fmt.Errorf("Proof result submitted suc: %v", txhash))
				return
			}
			if tryCount >= 3 {
				return
			}
			time.Sleep(configs.BlockInterval)
		}
	}
	return
}
