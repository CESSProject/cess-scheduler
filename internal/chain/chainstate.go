package chain

import (
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"
	"net/http"

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

	pub, err := tools.DecodeToPub(addr)
	if err != nil {
		return mdata, http.StatusBadRequest, errors.Wrapf(err, "[%v.%v:DecodeToPub]", State_Sminer, Sminer_MinerItems)
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerItems, pub)
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
func GetAllMinerDataOnChain(chainModule, chainModuleMethod string) ([]CessChain_AllMinerInfo, error) {
	var (
		err   error
		mdata []CessChain_AllMinerInfo
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
		return mdata, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", chainModule, chainModuleMethod)
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", chainModule, chainModuleMethod)
	}

	_, err = api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", chainModule, chainModuleMethod)
	}
	return mdata, nil
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
func GetFileMetaInfoOnChain(chainModule, chainModuleMethod, fileid string) (FileMetaInfo, error) {
	var (
		err   error
		mdata FileMetaInfo
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
		return mdata, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", chainModule, chainModuleMethod)
	}

	b, err := types.EncodeToBytes(fileid)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:EncodeToBytes]", chainModule, chainModuleMethod)
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, types.NewBytes(b))
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", chainModule, chainModuleMethod)
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", chainModule, chainModuleMethod)
	}
	if !ok {
		return mdata, errors.Errorf("[%v not folund]", fileid)
	}
	return mdata, nil
}

// Query Scheduler info
func GetSchedulerInfoOnChain(chainModule, chainModuleMethod string) ([]SchedulerInfo, error) {
	var (
		err   error
		mdata = make([]SchedulerInfo, 0)
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
		return mdata, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", chainModule, chainModuleMethod)
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", chainModule, chainModuleMethod)
	}

	_, err = api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", chainModule, chainModuleMethod)
	}
	return mdata, nil
}
