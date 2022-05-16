package chain

import (
	"cess-scheduler/configs"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"
	"net/http"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Get miner information on the chain
func GetMinerDataOnChain(addr string) (Chain_MinerItems, int32, error) {
	var (
		err   error
		mdata Chain_MinerItems
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_Sminer, Sminer_MinerItems, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, http.StatusInternalServerError, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", State_Sminer, Sminer_MinerItems)
	}

	pub, err := tools.DecodeToPub(addr, tools.ChainCessTestPrefix)
	if err != nil {
		return mdata, http.StatusBadRequest, errors.Wrapf(err, "[%v.%v:DecodeToPub]", State_Sminer, Sminer_MinerItems)
	}

	b, err := types.EncodeToBytes(types.NewAccountID(pub))
	if err != nil {
		return mdata, http.StatusBadRequest, errors.Wrapf(err, "[%v.%v:EncodeToBytes]", State_Sminer, Sminer_MinerItems)
	}
	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerItems, b)
	if err != nil {
		return mdata, http.StatusInternalServerError, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", State_Sminer, Sminer_MinerItems)
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, http.StatusInternalServerError, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", State_Sminer, Sminer_MinerItems)
	}
	if !ok {
		return mdata, http.StatusNotFound, errors.Errorf("[%v not found]", addr)
	}
	return mdata, http.StatusOK, nil
}

// Get all miner information on the cess chain
func GetAllMinerDataOnChain() ([]CessChain_AllMinerInfo, int, error) {
	var (
		err   error
		mdata []CessChain_AllMinerInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_Sminer, Sminer_AllMinerItems, err)
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
		return mdata, configs.Code_403, errors.New("No miners for storage")
	}
	return mdata, configs.Code_200, nil
}

// Get miner information on the cess chain
func GetMinerDetailsById(id uint64) (Chain_MinerDetails, int, error) {
	var (
		err   error
		mdata Chain_MinerDetails
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_Sminer, Sminer_MinerDetails, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(id)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	}
	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerDetails, b)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("Not found miner")
	}
	return mdata, configs.Code_200, nil
}

// Get unverified vpa or vpb on the cess chain
func GetUnverifiedVpaVpb(chainModule, chainModuleMethod string) ([]UnVerifiedVpaVpb, error) {
	var (
		err       error
		paramdata = make([]UnVerifiedVpaVpb, 0)
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", chainModule, chainModuleMethod, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", chainModule, chainModuleMethod)
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", chainModule, chainModuleMethod)
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", chainModule, chainModuleMethod)
	}
	return paramdata, nil
}

// Get unverified vpc on the cess chain
func GetUnverifiedVpc(chainModule, chainModuleMethod string) ([]UnVerifiedVpc, error) {
	var (
		err       error
		paramdata = make([]UnVerifiedVpc, 0)
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", chainModule, chainModuleMethod, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", chainModule, chainModuleMethod)
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", chainModule, chainModuleMethod)
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", chainModule, chainModuleMethod)
	}
	return paramdata, nil
}

// Get unverified vpd on the cess chain
func GetUnverifiedVpd(chainModule, chainModuleMethod string) ([]UnVerifiedVpd, error) {
	var (
		err       error
		paramdata = make([]UnVerifiedVpd, 0)
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", chainModule, chainModuleMethod, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", chainModule, chainModuleMethod)
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", chainModule, chainModuleMethod)
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", chainModule, chainModuleMethod)
	}
	return paramdata, nil
}

// Query file meta info
func GetFileMetaInfoOnChain(fileid string) (FileMetaInfo, int, error) {
	var (
		err   error
		mdata FileMetaInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_FileBank, FileMap_FileMetaInfo, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(fileid)
	if err != nil {
		return mdata, configs.Code_400, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileMap_FileMetaInfo, types.NewBytes(b))
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_400, errors.Errorf("[%v not folund]", fileid)
	}
	return mdata, configs.Code_200, nil
}

// Query Scheduler info
func GetSchedulerInfoOnChain() ([]SchedulerInfo, int, error) {
	var (
		err   error
		mdata = make([]SchedulerInfo, 0)
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_FileMap, FileMap_SchedulerInfo, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", State_FileMap, FileMap_SchedulerInfo)
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", State_FileMap, FileMap_SchedulerInfo)
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", State_FileMap, FileMap_SchedulerInfo)
	}
	if !ok {
		return mdata, configs.Code_404, errors.Errorf("[%v.%v:value is empty]", State_FileMap, FileMap_SchedulerInfo)
	}
	return mdata, configs.Code_200, nil
}

//
func GetSchedulerPukFromChain() (Chain_SchedulerPuk, int, error) {
	var (
		err  error
		data Chain_SchedulerPuk
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
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
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
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
	acc, err := tools.Encode(keyring.PublicKey, tools.SubstratePrefix)
	if err != nil {
		return "", errors.Wrap(err, "[Encode]")
	}
	return acc, nil
}

//
func GetFileRecoveryByAcc(prk string) ([]types.Bytes, int, error) {
	var (
		err  error
		data []types.Bytes
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
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
		return data, configs.Code_404, errors.New("public key not found")
	}
	return data, configs.Code_200, nil
}
