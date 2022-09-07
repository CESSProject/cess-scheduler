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

package chain

import (
	"strings"
	"time"

	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

func (c *chainClient) Register(stash, contact string) (string, error) {
	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	c.l.Lock()
	defer c.l.Unlock()

	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	stashPuk, err := utils.DecodePublicKeyOfCessAccount(stash)
	if err != nil {
		return txhash, errors.Wrap(err, "DecodePublicKeyOfCessAccount")
	}

	call, err := types.NewCall(
		c.metadata,
		tx_FileMap_Add_schedule,
		types.NewAccountID(stashPuk),
		types.Bytes([]byte(contact)),
	)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(call)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		state_System,
		system_Account,
		c.keyring.PublicKey,
	)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          c.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        c.runtimeVersion.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: c.runtimeVersion.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(c.keyring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		var tryCount = 0
		if !strings.Contains(err.Error(), "Priority is too low") {
			return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
		}
		for tryCount < 20 {
			o.Nonce = types.NewUCompactFromUInt(uint64(accountInfo.Nonce + types.NewU32(1)))
			// Sign the transaction
			err = ext.Sign(c.keyring, o)
			if err != nil {
				return txhash, errors.Wrap(err, "[Sign]")
			}
			sub, err = c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
			if err == nil {
				break
			}
			tryCount++
		}
	}
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(c.timeForBlockOut)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := CessEventRecords{}
				txhash, _ = types.EncodeToHex(status.AsInBlock)
				h, err := c.c.RPC.State.GetStorageRaw(c.keyEvents, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(c.metadata, &events)

				if len(events.FileMap_RegistrationScheduler) > 0 {
					if string(events.FileMap_RegistrationScheduler[0].Acc[:]) == string(c.keyring.PublicKey) {
						return txhash, nil
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "[sub]")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Update file meta information
func (c *chainClient) SubmitFileMeta(fid string, fsize uint64, user []byte, block []BlockInfo) (string, error) {
	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	c.l.Lock()
	defer c.l.Unlock()

	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	call, err := types.NewCall(
		c.metadata,
		tx_FileBank_Upload,
		types.NewBytes([]byte(fid)),
		types.U64(fsize),
		block,
		types.NewAccountID(user),
	)
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall")
	}

	ext := types.NewExtrinsic(call)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		state_System,
		system_Account,
		c.keyring.PublicKey,
	)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest err")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          c.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        c.runtimeVersion.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: c.runtimeVersion.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(c.keyring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "Sign")
	}

	// Do the transfer and track the actual status
	sub, err := c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		var tryCount = 0
		if !strings.Contains(err.Error(), "Priority is too low") {
			return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
		}
		for tryCount < 20 {
			o.Nonce = types.NewUCompactFromUInt(uint64(accountInfo.Nonce + types.NewU32(1)))
			// Sign the transaction
			err = ext.Sign(c.keyring, o)
			if err != nil {
				return txhash, errors.Wrap(err, "[Sign]")
			}
			sub, err = c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
			if err == nil {
				break
			}
			tryCount++
		}
	}
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(c.timeForBlockOut)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := CessEventRecords{}
				txhash, _ = types.EncodeToHex(status.AsInBlock)
				h, err := c.c.RPC.State.GetStorageRaw(c.keyEvents, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(c.metadata, &events)

				if len(events.FileBank_FileUpload) > 0 {
					return txhash, nil
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Update file meta information
func (c *chainClient) SubmitFillerMeta(miner_acc types.AccountID, info []FillerMetaInfo) (string, error) {
	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	c.l.Lock()
	defer c.l.Unlock()

	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	call, err := types.NewCall(c.metadata, tx_FileBank_UploadFiller, miner_acc, info)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(call)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		state_System,
		system_Account,
		c.keyring.PublicKey,
	)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          c.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        c.runtimeVersion.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: c.runtimeVersion.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(c.keyring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		var tryCount = 0
		if !strings.Contains(err.Error(), "Priority is too low") {
			return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
		}
		for tryCount < 20 {
			o.Nonce = types.NewUCompactFromUInt(uint64(accountInfo.Nonce + types.NewU32(1)))
			// Sign the transaction
			err = ext.Sign(c.keyring, o)
			if err != nil {
				return txhash, errors.Wrap(err, "[Sign]")
			}
			sub, err = c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
			if err == nil {
				break
			}
			tryCount++
		}
	}
	if err != nil {
		return "", errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(c.timeForBlockOut)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := CessEventRecords{}
				txhash, _ = types.EncodeToHex(status.AsInBlock)
				h, err := c.c.RPC.State.GetStorageRaw(c.keyEvents, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(c.metadata, &events)

				if len(events.FileBank_FillerUpload) > 0 {
					return txhash, nil
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

//
func (c *chainClient) SubmitProofResults(data []ProofResult) (string, error) {
	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	c.l.Lock()
	defer c.l.Unlock()

	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	call, err := types.NewCall(c.metadata, tx_SegmentBook_VerifyProof, data)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(call)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		state_System,
		system_Account,
		c.keyring.PublicKey,
	)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          c.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        c.runtimeVersion.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: c.runtimeVersion.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(c.keyring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		var tryCount = 0
		if !strings.Contains(err.Error(), "Priority is too low") {
			return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
		}
		for tryCount < 20 {
			o.Nonce = types.NewUCompactFromUInt(uint64(accountInfo.Nonce + types.NewU32(1)))
			// Sign the transaction
			err = ext.Sign(c.keyring, o)
			if err != nil {
				return txhash, errors.Wrap(err, "[Sign]")
			}
			sub, err = c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
			if err == nil {
				break
			}
			tryCount++
		}
	}
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}

	defer sub.Unsubscribe()
	timeout := time.After(c.timeForBlockOut)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := CessEventRecords{}
				txhash, _ = types.EncodeToHex(status.AsInBlock)
				h, err := c.c.RPC.State.GetStorageRaw(c.keyEvents, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "[GetStorageRaw]")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(c.metadata, &events)

				if len(events.SegmentBook_VerifyProof) > 0 {
					return txhash, nil
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

func (c *chainClient) Update(contact string) (string, error) {
	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	c.l.Lock()
	defer c.l.Unlock()

	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	call, err := types.NewCall(
		c.metadata,
		tx_FileMap_UpdateScheduler,
		types.Bytes([]byte(contact)),
	)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(call)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		state_System,
		system_Account,
		c.keyring.PublicKey,
	)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          c.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        c.runtimeVersion.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: c.runtimeVersion.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(c.keyring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		var tryCount = 0
		if !strings.Contains(err.Error(), "Priority is too low") {
			return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
		}
		for tryCount < 20 {
			o.Nonce = types.NewUCompactFromUInt(uint64(accountInfo.Nonce + types.NewU32(1)))
			// Sign the transaction
			err = ext.Sign(c.keyring, o)
			if err != nil {
				return txhash, errors.Wrap(err, "[Sign]")
			}
			sub, err = c.c.RPC.Author.SubmitAndWatchExtrinsic(ext)
			if err == nil {
				break
			}
			tryCount++
		}
	}
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(c.timeForBlockOut)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := CessEventRecords{}
				txhash, _ = types.EncodeToHex(status.AsInBlock)
				h, err := c.c.RPC.State.GetStorageRaw(c.keyEvents, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "[GetStorageRaw]")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(c.metadata, &events)

				if len(events.FileMap_UpdateScheduler) > 0 {
					if string(events.FileMap_UpdateScheduler[0].Acc[:]) == string(c.keyring.PublicKey) {
						return txhash, nil
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}
