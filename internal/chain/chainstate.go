package chain

import (
	"scheduler-mining/internal/logger"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

type CessChain_MinerItems struct {
	Peerid      types.U64       `json:"peerid"`
	Beneficiary types.AccountID `json:"beneficiary"`
	Ip          types.U32       `json:"ip"`
	Collaterals types.U128      `json:"collaterals"`
	Earnings    types.U128      `json:"earnings"`
	Locked      types.U128      `json:"locked"`
}

type CessChain_AllMinerItems struct {
	Peerid   types.U64  `json:"peerid"`
	Ip       types.U32  `json:"ip"`
	Port     types.U32  `json:"port"`
	FilePort types.U32  `json:"fileport"`
	Power    types.U128 `json:"power"`
	Space    types.U128 `json:"space"`
}

type SegmentInfo struct {
	Segment_index types.U64 `json:"segment_index"`
	//valid space
	Power types.U128 `json:"power"`
	//used space
	Space types.U128 `json:"space"`
}

type ParamInfo struct {
	Peer_id    types.U64 `json:"peer_id"`
	Segment_id types.U64 `json:"segment_id"`
	Rand       types.U32 `json:"rand"`
}

type IpostParaInfo struct {
	Peer_id    types.U64   `json:"peer_id"`
	Segment_id types.U64   `json:"segment_id"`
	Size_type  types.U128  `json:"size_type"`
	Sealed_cid types.Bytes `json:"sealed_cid"`
}

type UnVerifiedVpaVpb struct {
	Accountid  types.AccountID `json:"acc"`
	Peer_id    types.U64       `json:"peer_id"`
	Segment_id types.U64       `json:"segment_id"`
	Proof      types.Bytes     `json:"proof"`
	Sealed_cid types.Bytes     `json:"sealed_cid"`
	Rand       types.U32       `json:"rand"`
	Size_type  types.U128      `json:"size_type"`
}

type UnVerifiedVpc struct {
	Accountid    types.AccountID `json:"acc"`
	Peer_id      types.U64       `json:"peer_id"`
	Segment_id   types.U64       `json:"segment_id"`
	Proof        []types.Bytes   `json:"proof"`
	Sealed_cid   []types.Bytes   `json:"sealed_cid"`
	Unsealed_cid []types.Bytes   `json:"unsealed_cid"`
	Rand         types.U32       `json:"rand"`
	Size_type    types.U128      `json:"size_type"`
}

type UnVerifiedVpd struct {
	Accountid  types.AccountID `json:"acc"`
	Peer_id    types.U64       `json:"peer_id"`
	Segment_id types.U64       `json:"segment_id"`
	Proof      []types.Bytes   `json:"proof"`
	Sealed_cid []types.Bytes   `json:"sealed_cid"`
	Rand       types.U32       `json:"rand"`
	Size_type  types.U128      `json:"size_type"`
}

// Get miner information on the cess chain
func GetMinerDataOnChain(api *gsrpc.SubstrateAPI, identifyAccountPhrase, chainModule, chainModuleMethod string) (CessChain_MinerItems, error) {
	var (
		err   error
		mdata CessChain_MinerItems
	)
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return mdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, account.PublicKey)
	if err != nil {
		return mdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return mdata, nil
}

// Get all miner information on the cess chain
func GetAllMinerDataOnChain(api *gsrpc.SubstrateAPI, chainModule, chainModuleMethod string) ([]CessChain_AllMinerItems, error) {
	var (
		err   error
		mdata []CessChain_AllMinerItems
	)
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return mdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return mdata, nil
}

// Get segment number on the cess chain
func GetSegNumOnChain(api *gsrpc.SubstrateAPI, identifyAccountPhrase, chainModule, chainModuleMethod string) (SegmentInfo, error) {
	var (
		err     error
		ok      bool
		segdata SegmentInfo
	)
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return segdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return segdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, account.PublicKey)
	if err != nil {
		return segdata, errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &segdata)
	if err != nil {
		return segdata, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return segdata, errors.New("segment data is empty")
	}
	return segdata, nil
}

// Get unverified vpa on the cess chain
func GetUnverifiedVpaVpb(api *gsrpc.SubstrateAPI, chainModule, chainModuleMethod string) ([]UnVerifiedVpaVpb, error) {
	var (
		err       error
		paramdata = make([]UnVerifiedVpaVpb, 0)
	)
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return paramdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return paramdata, nil
}

// Get unverified vpa on the cess chain
func GetUnverifiedVpc(api *gsrpc.SubstrateAPI, chainModule, chainModuleMethod string) ([]UnVerifiedVpc, error) {
	var (
		err       error
		paramdata = make([]UnVerifiedVpc, 0)
	)
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return paramdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return paramdata, nil
}

// Get unverified vpa on the cess chain
func GetUnverifiedVpd(api *gsrpc.SubstrateAPI, chainModule, chainModuleMethod string) ([]UnVerifiedVpd, error) {
	var (
		err       error
		paramdata = make([]UnVerifiedVpd, 0)
	)
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod)
	if err != nil {
		return paramdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return paramdata, nil
}
