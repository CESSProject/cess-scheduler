package chain

import (
	. "cess-scheduler/internal/logger"

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

type CessChain_AllMinerInfo struct {
	Peerid types.U64   `json:"peerid"`
	Ip     types.Bytes `json:"ip"`
	Power  types.U128  `json:"power"`
	Space  types.U128  `json:"space"`
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

type FileMetaInfo struct {
	//FileId      types.Bytes         `json:"acc"`         //File id
	File_name   types.Bytes         `json:"file_name"`   //File name
	FileSize    types.U128          `json:"file_size"`   //File size
	FileHash    types.Bytes         `json:"file_hash"`   //File hash
	Public      types.Bool          `json:"public"`      //Public or not
	UserAddr    types.AccountID     `json:"user_addr"`   //Upload user's address
	FileState   types.Bytes         `json:"file_state"`  //File state
	Backups     types.U8            `json:"backups"`     //Number of backups
	Downloadfee types.U128          `json:"downloadfee"` //Download fee
	FileDupl    []FileDuplicateInfo `json:"file_dupl"`   //File backup information list
}

type FileDuplicateInfo struct {
	DuplId    types.Bytes     `json:"dupl_id"`    //Backup id
	RandKey   types.Bytes     `json:"rand_key"`   //Random key
	SliceNum  types.U16       `json:"slice_num"`  //Number of slices
	FileSlice []FileSliceInfo `json:"file_slice"` //Slice information list
}

type FileSliceInfo struct {
	SliceId   types.Bytes   `json:"slice_id"`   //Slice id
	SliceSize types.U32     `json:"slice_size"` //Slice size
	SliceHash types.Bytes   `json:"slice_hash"` //Slice hash
	FileShard FileShardInfo `json:"file_shard"` //Shard information
}

type FileShardInfo struct {
	DataShardNum  types.U8      `json:"data_shard_num"`  //Number of data shard
	RedunShardNum types.U8      `json:"redun_shard_num"` //Number of redundant shard
	ShardHash     []types.Bytes `json:"shard_hash"`      //Shard hash list
	ShardAddr     []types.Bytes `json:"shard_addr"`      //Store miner service addr list
	Peerid        []types.U64   `json:"wallet_addr"`     //Store miner wallet addr list
}

type SchedulerInfo struct {
	Ip  types.Bytes
	Acc types.AccountID
}

type CessChain_EtcdItems struct {
	Ip types.Bytes `json:"ip"`
}

// Get miner information on the cess chain
func GetMinerDataOnChain(identifyAccountPhrase, chainModule, chainModuleMethod string) (CessChain_MinerItems, error) {
	var (
		err   error
		mdata CessChain_MinerItems
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

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:KeyringPairFromSecret]", chainModule, chainModuleMethod)
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, account.PublicKey)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", chainModule, chainModuleMethod)
	}

	_, err = api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", chainModule, chainModuleMethod)
	}
	return mdata, nil
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

	_, err = api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", chainModule, chainModuleMethod)
	}
	//fmt.Println(mdata)
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
