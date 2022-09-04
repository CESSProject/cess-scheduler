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
	"time"

	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

func (c *chainClient) Register(stash, contact string) (string, error) {
	var txhash string
	var accountInfo types.AccountInfo

	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	stashPuk, err := utils.DecodePublicKeyOfCessAccount(stash)
	if err != nil {
		return txhash, errors.Wrap(err, "DecodePublicKeyOfCessAccount")
	}

	call, err := types.NewCall(
		c.metadata,
		Tx_FileMap_Add_schedule,
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
		State_System,
		System_Account,
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
func (c *chainClient) SubmitFileMeta(fid string, fsize uint64, user []byte, chunk []BlockInfo) (string, error) {
	var (
		txhash      string
		accountInfo types.AccountInfo
	)
	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	call, err := types.NewCall(
		c.metadata,
		Tx_FileBank_Upload,
		types.NewBytes([]byte(fid)),
		types.U64(fsize),
		chunk,
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
		State_System,
		System_Account,
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
	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	call, err := types.NewCall(c.metadata, Tx_FileBank_UploadFiller, miner_acc, info)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(call)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_System,
		System_Account,
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
	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	call, err := types.NewCall(c.metadata, Tx_SegmentBook_VerifyProof, data)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(call)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_System,
		System_Account,
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

//
// func ClearRecoveredFileNoChain(signaturePrk string, duplid types.Bytes) (int, error) {
// 	var (
// 		err         error
// 		accountInfo types.AccountInfo
// 	)
// 	api := SubApi.getApi()
// 	defer func() {
// 		SubApi.free()
// 		if err := recover(); err != nil {
// 			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
// 		}
// 	}()

// 	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
// 	if err != nil {
// 		return configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
// 	}

// 	meta, err := api.RPC.State.GetMetadataLatest()
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
// 	}

// 	c, err := types.NewCall(meta, FileBank_ClearRecoveredFile, duplid)
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[NewCall]")
// 	}

// 	ext := types.NewExtrinsic(c)
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[NewExtrinsic]")
// 	}

// 	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[GetBlockHash]")
// 	}

// 	rv, err := api.RPC.State.GetRuntimeVersionLatest()
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[GetRuntimeVersionLatest]")
// 	}

// 	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Account]")
// 	}

// 	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Events]")
// 	}

// 	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
// 	}
// 	if !ok {
// 		return configs.Code_500, errors.New("[GetStorageLatest return value is empty]")
// 	}

// 	o := types.SignatureOptions{
// 		BlockHash:          genesisHash,
// 		Era:                types.ExtrinsicEra{IsMortalEra: false},
// 		GenesisHash:        genesisHash,
// 		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
// 		SpecVersion:        rv.SpecVersion,
// 		Tip:                types.NewUCompactFromUInt(0),
// 		TransactionVersion: rv.TransactionVersion,
// 	}

// 	// Sign the transaction
// 	err = ext.Sign(keyring, o)
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[Sign]")
// 	}

// 	// Do the transfer and track the actual status
// 	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
// 	if err != nil {
// 		return configs.Code_500, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
// 	}
// 	defer sub.Unsubscribe()
// 	var head *types.Header
// 	t := tools.RandomInRange(10000000, 99999999)
// 	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
// 	for {
// 		select {
// 		case status := <-sub.Chan():
// 			if status.IsInBlock {
// 				events := MyEventRecords{}
// 				head, err = api.RPC.Chain.GetHeader(status.AsInBlock)
// 				if err == nil {
// 					Com.Sugar().Infof("[T:%v] [BN:%v]", t, head.Number)
// 				} else {
// 					Com.Sugar().Infof("[T:%v] [BH:%#x]", t, status.AsInBlock)
// 				}
// 				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
// 				if err != nil {
// 					return configs.Code_600, errors.Wrapf(err, "[T:%v]", t)
// 				}
// 				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
// 				if err != nil {
// 					Com.Sugar().Errorf("[T:%v]Decode event err:%v", t, err)
// 				}
// 				if events.FileBank_ClearInvalidFile != nil {
// 					for i := 0; i < len(events.FileBank_ClearInvalidFile); i++ {
// 						if events.FileBank_ClearInvalidFile[i].Acc == types.NewAccountID(keyring.PublicKey) {
// 							Com.Sugar().Infof("[T:%v] Submit prove success", t)
// 							return configs.Code_200, nil
// 						}
// 					}
// 					return configs.Code_600, errors.Errorf("[T:%v] events.FileBank_ClearInvalidFile data err", t)
// 				}
// 				return configs.Code_600, errors.Errorf("[T:%v] events.FileBank_ClearInvalidFile not found", t)
// 			}
// 		case err = <-sub.Err():
// 			return configs.Code_500, err
// 		case <-timeout:
// 			return configs.Code_500, errors.New("Timeout")
// 		}
// 	}
// }

func (c *chainClient) Update(contact string) (string, error) {
	var (
		txhash      string
		accountInfo types.AccountInfo
	)
	if !c.IsChainClientOk() {
		return txhash, errors.New("rpc connection failed")
	}

	call, err := types.NewCall(
		c.metadata,
		Tx_FileMap_UpdateScheduler,
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
		State_System,
		System_Account,
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
