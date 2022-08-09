package chain

import (
	"cess-scheduler/configs"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

type CessInfo struct {
	RpcAddr                string
	IdentifyAccountPhrase  string
	IncomeAccountPublicKey string
	TransactionName        string
	ChainModule            string
	ChainModuleMethod      string
}

func RegisterToChain(transactionPrK, stash_acc, ipAddr string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var txhash string
	var accountInfo types.AccountInfo

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	bytes, err := tools.DecodeToPub(stash_acc, tools.ChainCessTestPrefix)
	if err != nil {
		return txhash, errors.Wrap(err, "DecodeToPub")
	}

	c, err := types.NewCall(meta, ChainTx_FileMap_Add_schedule, types.NewAccountID(bytes), types.Bytes([]byte(ipAddr)))
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", configs.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	kring, err := GetKeyring()
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "Sign")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Infof("[%v]Decode event err:%v", txhash, err)
				}

				if len(events.FileMap_RegistrationScheduler) > 0 {
					for i := 0; i < len(events.FileMap_RegistrationScheduler); i++ {
						if string(events.FileMap_RegistrationScheduler[i].Acc[:]) == string(configs.PublicKey) {
							return txhash, nil
						}
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Update file meta information
func PutMetaInfoToChain(transactionPrK, fid string, fsize uint64, user []byte, chunk []ChunkInfo) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var txhash string
	var accountInfo types.AccountInfo

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(
		meta,
		Tx_FileBank_Upload,
		types.NewBytes([]byte(fid)),
		types.U64(fsize),
		chunk,
		types.NewAccountID(user),
	)
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", configs.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest err")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	kring, err := GetKeyring()
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "Sign")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				Uld.Sugar().Infof("[%v] FileMeta On-chain hash: %v", fid, txhash)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Infof("[%v]Decode event err:%v", txhash, err)
				}

				if len(events.FileBank_FileUpload) > 0 {
					for i := 0; i < len(events.FileBank_FileUpload); i++ {
						if string(events.FileBank_FileUpload[i].Acc[:]) == string(user) {
							return txhash, nil
						}
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, err
		case <-timeout:
			return txhash, errors.New("timeout")
		}
	}
}

// Update file meta information
func PutSpaceTagInfoToChain(transactionPrK string, miner_acc types.AccountID, info []SpaceFileInfo) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var txhash string
	var accountInfo types.AccountInfo

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, ChainTx_FileBank_UploadFiller, miner_acc, info)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", configs.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	kring, err := GetKeyring()
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Infof("[%v]Decode event err:%v", txhash, err)
				}

				if len(events.FileBank_FillerUpload) > 0 {
					for i := 0; i < len(events.FileBank_FillerUpload); i++ {
						if string(events.FileBank_FillerUpload[i].Acc[:]) == string(configs.PublicKey) {
							return txhash, nil
						}
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

//
func PutProofResult(signaturePrk string, data []VerifyResult) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var txhash string
	var accountInfo types.AccountInfo

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, SegmentBook_VerifyProof, data)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", configs.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	kring, err := GetKeyring()
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Infof("[%v]Decode event err:%v", txhash, err)
				}

				if len(events.SegmentBook_VerifyProof) > 0 {
					for i := 0; i < len(events.SegmentBook_VerifyProof); i++ {
						if string(events.SegmentBook_VerifyProof[i].Miner[:]) == string(data[0].Miner_pubkey[:]) {
							return txhash, nil
						}
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
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

func UpdatePublicIp(transactionPrK, ipAddr string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var txhash string
	var accountInfo types.AccountInfo

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, FileMap_UpdateScheduler, types.Bytes([]byte(ipAddr)))
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", configs.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest")
	}
	if !ok {
		return txhash, errors.New(ERR_Empty)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	kring, err := GetKeyring()
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "Sign")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Infof("[%v]Decode event err:%v", txhash, err)
				}

				if len(events.FileMap_UpdateScheduler) > 0 {
					for i := 0; i < len(events.FileMap_UpdateScheduler); i++ {
						if string(events.FileMap_UpdateScheduler[i].Acc[:]) == string(configs.PublicKey) {
							return txhash, nil
						}
					}
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}
