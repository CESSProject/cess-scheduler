package chain

import (
	"cess-scheduler/configs"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Get miner information on the chain
func GetMinerInfo(pubkey types.AccountID) (MinerInfo, int, error) {
	var (
		err   error
		mdata MinerInfo
	)
	api := SubApi.getApi()
	defer func() {
		SubApi.free()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(pubkey)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerItems, b)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("[value is empty]")
	}
	return mdata, configs.Code_200, nil
}

// Get all miner information on the cess chain
func GetAllMinerDataOnChain() ([]types.AccountID, int, error) {
	var (
		err   error
		mdata []types.AccountID
	)
	api := SubApi.getApi()
	defer func() {
		SubApi.free()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_AllMinerItems)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("[value is empty]")
	}
	return mdata, configs.Code_200, nil
}

// Query file meta info
func GetFileMetaInfo(fid string) (FileMetaInfo, int, error) {
	var (
		err   error
		mdata FileMetaInfo
	)
	api := SubApi.getApi()
	defer func() {
		SubApi.free()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(fid)
	if err != nil {
		return mdata, configs.Code_400, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileMap_FileMetaInfo, b)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("[Not found]")
	}
	return mdata, configs.Code_200, nil
}

// Query Scheduler info
func GetSchedulerInfoOnChain() ([]SchedulerInfo, int, error) {
	var (
		err   error
		mdata = make([]SchedulerInfo, 0)
	)
	api := SubApi.getApi()
	defer func() {
		SubApi.free()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("[Not found]")
	}
	return mdata, configs.Code_200, nil
}

//
func GetSchedulerPukFromChain() (Chain_SchedulerPuk, int, error) {
	var (
		err  error
		data Chain_SchedulerPuk
	)
	api := SubApi.getApi()
	defer func() {
		SubApi.free()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerPuk)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("public key not found")
	}
	return data, configs.Code_200, nil
}

//
func GetProofsFromChain(prk string) ([]Chain_Proofs, int, error) {
	var (
		err  error
		data []Chain_Proofs
	)
	api := SubApi.getApi()
	defer func() {
		SubApi.free()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return data, configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_SegmentBook, SegmentBook_UnVerifyProof, keyring.PublicKey)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("public key not found")
	}
	return data, configs.Code_200, nil
}

//
func GetAddressByPrk(prk string) (string, error) {
	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return "", errors.Wrap(err, "[KeyringPairFromSecret]")
	}
	acc, err := tools.EncodeToCESSAddr(keyring.PublicKey)
	if err != nil {
		return "", errors.Wrap(err, "[Encode]")
	}
	return acc, nil
}

//
func GetPublicKeyByPrk(prk string) ([]byte, error) {
	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return nil, errors.Wrap(err, "[KeyringPairFromSecret]")
	}
	return keyring.PublicKey, nil
}

//
func GetFileRecoveryByAcc(prk string) ([]types.Bytes, int, error) {
	var (
		err  error
		data []types.Bytes
	)
	api := SubApi.getApi()
	defer func() {
		SubApi.free()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return data, configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_FileRecovery, keyring.PublicKey)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

//
func GetSpacePackageInfo(puk types.AccountID) (SpacePackage, int, error) {
	var (
		err   error
		mdata SpacePackage
	)
	api := SubApi.getApi()
	defer func() {
		SubApi.free()
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(puk)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_PurchasedPackage, b)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("value is empty")
	}
	return mdata, configs.Code_200, nil
}
