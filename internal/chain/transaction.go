package chain

import (
	"cess-scheduler/configs"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"
	"encoding/json"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
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

func RegisterToChain(transactionPrK, TransactionName, stash_acc, ipAddr string) (bool, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}
	var pre []byte
	if configs.NewTestAddr {
		pre = tools.ChainCessTestPrefix
	} else {
		pre = tools.SubstratePrefix
	}
	bytes, err := tools.DecodeToPub(stash_acc, pre)
	if err != nil {
		return false, errors.Wrap(err, "DecodeToPub")
	}

	c, err := types.NewCall(meta, TransactionName, types.NewAccountID(bytes), types.Bytes([]byte(ipAddr)))
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
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

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return false, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return false, err
					}
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}
				if events.FileMap_RegistrationScheduler != nil {
					for i := 0; i < len(events.FileMap_RegistrationScheduler); i++ {
						if events.FileMap_RegistrationScheduler[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return true, nil
						}
					}
					if head != nil {
						return false, errors.Errorf("[%v]events.FileMap_RegistrationScheduler data err", head.Number)
					} else {
						return false, errors.New("events.FileMap_RegistrationScheduler data err")
					}
				}
				if head != nil {
					return false, errors.Errorf("[%v]events.FileMap_RegistrationScheduler not found", head.Number)
				} else {
					return false, errors.New("events.FileMap_RegistrationScheduler not found")
				}
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.Errorf("[%v] tx timeout", TransactionName)
		}
	}
}

// Update file meta information
func PutMetaInfoToChain(transactionPrK, fid string, info []FileDuplicateInfo) (bool, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, ChainTx_FileBank_PutMetaInfo, types.Bytes([]byte(fid)), info)
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
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

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return false, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return false, err
					}
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}

				if events.FileBank_FileUpdate != nil {
					for i := 0; i < len(events.FileBank_FileUpdate); i++ {
						if string(events.FileBank_FileUpdate[i].Fileid) == string(fid) {
							return true, nil
						}
					}
					if head != nil {
						return false, errors.Errorf("[%v]events.FileBank_FileUpdate data err", head.Number)
					} else {
						return false, errors.New("events.FileBank_FileUpdate data err")
					}
				}
				if head != nil {
					return false, errors.Errorf("[%v]events.FileBank_FileUpdate not found", head.Number)
				} else {
					return false, errors.New("events.FileBank_FileUpdate not found")
				}
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.Errorf("[%v] tx timeout", ChainTx_FileBank_PutMetaInfo)
		}
	}
}

// Update file meta information
func PutSpaceTagInfoToChain(transactionPrK string, mid types.U64, info []SpaceFileInfo) (bool, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}

	// b, err := types.EncodeToBytes(mid)
	// if err != nil {
	// 	return false, errors.Wrap(err, "EncodeToBytes err")
	// }
	c, err := types.NewCall(meta, ChainTx_FileBank_UploadFiller, mid, info)
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
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

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return false, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return false, err
					}
				}
				//types.EventRecordsRaw(key).DecodeEventRecords(meta, &events)
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}

				if events.FileBank_FillerUpload != nil {
					for i := 0; i < len(events.FileBank_FillerUpload); i++ {
						if events.FileBank_FillerUpload[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return true, nil
						}
					}
					if head != nil {
						return false, errors.Errorf("[%v]events.FileBank_FillerUpload data err", head.Number)
					} else {
						return false, errors.New("events.FileBank_FillerUpload data err")
					}
				}
				if head != nil {
					return false, errors.Errorf("[%v]events.FileBank_FillerUpload not found", head.Number)
				} else {
					return false, errors.New("events.FileBank_FillerUpload not found")
				}
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.Errorf("[%v] tx timeout", ChainTx_FileBank_UploadFiller)
		}
	}
}

type faucet struct {
	Ans    answer `json:"Result"`
	Status string `json:"Status"`
}
type answer struct {
	Err       string `json:"Err"`
	AsInBlock bool   `json:"AsInBlock"`
}

// obtain tCESS
func ObtainFromFaucet(faucetaddr, pbk string) error {
	var ob = struct {
		Address string `json:"Address"`
	}{
		pbk,
	}
	var res faucet
	resp, err := tools.Post(faucetaddr, ob)
	if err != nil {
		return err
	}
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return err
	}
	if res.Ans.Err != "" {
		return err
	}

	if res.Ans.AsInBlock {
		return nil
	} else {
		return errors.New("The address has been picked up today, please come back after 1 day.")
	}
}

//
func PutProofResult(signaturePrk string, id types.U64, fid types.Bytes, result bool) (int, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
	if err != nil {
		return configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, SegmentBook_VerifyProof, id, fid, types.Bool(result))
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetBlockHash]")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetRuntimeVersionLatest]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Events]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return configs.Code_500, errors.New("[GetStorageLatest return value is empty]")
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

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	t := tools.RandomInRange(10000000, 99999999)
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, err = api.RPC.Chain.GetHeader(status.AsInBlock)
				if err == nil {
					Out.Sugar().Infof("[T:%v] [BN:%v]", t, head.Number)
				} else {
					Out.Sugar().Infof("[T:%v] [BH:%#x]", t, status.AsInBlock)
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return configs.Code_600, errors.Wrapf(err, "[T:%v]", t)
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("[T:%v]Decode event err:%v", t, err)
				}
				if events.SegmentBook_VerifyProof != nil {
					for i := 0; i < len(events.SegmentBook_VerifyProof); i++ {
						if events.SegmentBook_VerifyProof[i].PeerId == types.U64(id) {
							Out.Sugar().Infof("[T:%v] Submit prove success", t)
							return configs.Code_200, nil
						}
					}
					return configs.Code_600, errors.Errorf("[T:%v] events.SegmentBook_VerifyProof data err", t)
				}
				return configs.Code_600, errors.Errorf("[T:%v] events.SegmentBook_VerifyProof not found", t)
			}
		case err = <-sub.Err():
			return configs.Code_500, err
		case <-timeout:
			return configs.Code_500, errors.New("Timeout")
		}
	}
}

//
func ClearRecoveredFileNoChain(signaturePrk string, duplid types.Bytes) (int, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
	if err != nil {
		return configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, FileBank_ClearRecoveredFile, duplid)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetBlockHash]")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetRuntimeVersionLatest]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Events]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return configs.Code_500, errors.New("[GetStorageLatest return value is empty]")
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

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	t := tools.RandomInRange(10000000, 99999999)
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, err = api.RPC.Chain.GetHeader(status.AsInBlock)
				if err == nil {
					Out.Sugar().Infof("[T:%v] [BN:%v]", t, head.Number)
				} else {
					Out.Sugar().Infof("[T:%v] [BH:%#x]", t, status.AsInBlock)
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return configs.Code_600, errors.Wrapf(err, "[T:%v]", t)
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("[T:%v]Decode event err:%v", t, err)
				}
				if events.FileBank_ClearInvalidFile != nil {
					for i := 0; i < len(events.FileBank_ClearInvalidFile); i++ {
						if events.FileBank_ClearInvalidFile[i].Acc == types.NewAccountID(keyring.PublicKey) {
							Out.Sugar().Infof("[T:%v] Submit prove success", t)
							return configs.Code_200, nil
						}
					}
					return configs.Code_600, errors.Errorf("[T:%v] events.FileBank_ClearInvalidFile data err", t)
				}
				return configs.Code_600, errors.Errorf("[T:%v] events.FileBank_ClearInvalidFile not found", t)
			}
		case err = <-sub.Err():
			return configs.Code_500, err
		case <-timeout:
			return configs.Code_500, errors.New("Timeout")
		}
	}
}
