package chain

import (
	"cess-scheduler/configs"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"
	"encoding/json"
	"fmt"
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

func RegisterToChain(transactionPrK, TransactionName, stash_acc, ipAddr string) (string, int, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetMetadataLatest err")
	}

	bytes, err := tools.DecodeToPub(stash_acc, tools.ChainCessTestPrefix)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "DecodeToPub")
	}

	c, err := types.NewCall(meta, TransactionName, types.NewAccountID(bytes), types.Bytes([]byte(ipAddr)))
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return "", configs.Code_500, errors.New("GetStorageLatest return value is empty")
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
		return "", configs.Code_500, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash := fmt.Sprintf("%#x", status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, configs.Code_600, err
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Infof("[%v] Decode event err: %v", txhash, err)
				}

				for i := 0; i < len(events.FileMap_RegistrationScheduler); i++ {
					if events.FileMap_RegistrationScheduler[i].Acc == types.NewAccountID(keyring.PublicKey) {
						return txhash, configs.Code_200, nil
					}
				}

				return txhash, configs.Code_600, errors.New("events.FileMap_RegistrationScheduler not found")

			}
		case err = <-sub.Err():
			return "", configs.Code_500, err
		case <-timeout:
			return "", configs.Code_500, errors.Errorf("[%v] tx timeout", TransactionName)
		}
	}
}

// Update file meta information
func PutMetaInfoToChain(transactionPrK, fid string, fsize uint64, user []byte, chunk []ChunkInfo) (string, error) {
	var (
		err         error
		txhash      string
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return txhash, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetMetadataLatest err")
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
		return txhash, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return txhash, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return txhash, errors.New("GetStorageLatest return value is empty")
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
		return txhash, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ = types.EncodeToHexString(status.AsInBlock)
				return txhash, nil
			}
		case err = <-sub.Err():
			return txhash, err
		case <-timeout:
			return txhash, errors.New("timeout")
		}
	}
}

// Update file meta information
func PutSpaceTagInfoToChain(transactionPrK string, miner_acc []byte, info []SpaceFileInfo) (string, int, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, ChainTx_FileBank_UploadFiller, types.NewAccountID(miner_acc), info)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[GetBlockHash]")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[GetRuntimeVersionLatest]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[CreateStorageKey System  Account]")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Events]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return "", configs.Code_500, errors.New("[GetStorageLatest value is empty]")
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
		return "", configs.Code_500, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash := fmt.Sprintf("%#x", status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, configs.Code_600, err
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Errorf("[%v]Decode event err:%v", txhash, err)
				}

				for i := 0; i < len(events.FileBank_FillerUpload); i++ {
					if events.FileBank_FillerUpload[i].Acc == types.NewAccountID(keyring.PublicKey) {
						return txhash, configs.Code_200, nil
					}
				}

				return txhash, configs.Code_600, errors.Errorf("events.FileBank_FillerUpload not found")
			}
		case err = <-sub.Err():
			return "", configs.Code_500, err
		case <-timeout:
			return "", configs.Code_500, errors.New("Timeout")
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
func PutProofResult(signaturePrk string, data []VerifyResult) (int, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
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

	c, err := types.NewCall(meta, SegmentBook_VerifyProof, data)
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
					Com.Sugar().Infof("[T:%v] [BN:%v]", t, head.Number)
				} else {
					Com.Sugar().Infof("[T:%v] [BH:%#x]", t, status.AsInBlock)
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return configs.Code_600, errors.Wrapf(err, "[T:%v]", t)
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Errorf("[T:%v]Decode event err:%v", t, err)
				}
				if len(events.SegmentBook_VerifyProof) > 0 {
					Com.Sugar().Infof("[T:%v] Submit prove success", t)
					return configs.Code_200, nil
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
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
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
					Com.Sugar().Infof("[T:%v] [BN:%v]", t, head.Number)
				} else {
					Com.Sugar().Infof("[T:%v] [BH:%#x]", t, status.AsInBlock)
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return configs.Code_600, errors.Wrapf(err, "[T:%v]", t)
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Errorf("[T:%v]Decode event err:%v", t, err)
				}
				if events.FileBank_ClearInvalidFile != nil {
					for i := 0; i < len(events.FileBank_ClearInvalidFile); i++ {
						if events.FileBank_ClearInvalidFile[i].Acc == types.NewAccountID(keyring.PublicKey) {
							Com.Sugar().Infof("[T:%v] Submit prove success", t)
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

func UpdatePublicIp(transactionPrK, ipAddr string) (string, int, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, FileMap_UpdateScheduler, types.Bytes([]byte(ipAddr)))
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return "", configs.Code_500, errors.New("GetStorageLatest return value is empty")
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
		return "", configs.Code_500, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash := fmt.Sprintf("%#x", status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, configs.Code_600, err
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Com.Sugar().Infof("[%v] Decode event err: %v", txhash, err)
				}

				for i := 0; i < len(events.FileMap_RegistrationScheduler); i++ {
					if events.FileMap_RegistrationScheduler[i].Acc == types.NewAccountID(keyring.PublicKey) {
						return txhash, configs.Code_200, nil
					}
				}

				return txhash, configs.Code_600, errors.New("events.FileMap_RegistrationScheduler not found")

			}
		case err = <-sub.Err():
			return "", configs.Code_500, err
		case <-timeout:
			return "", configs.Code_500, errors.Errorf("tx timeout")
		}
	}
}
