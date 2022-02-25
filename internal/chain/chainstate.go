package chain

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"scheduler-mining/internal/logger"
	"strings"

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
			logger.ErrLogger.Sugar().Errorf("[panic] [%v.%v] [err:%v]", chainModule, chainModuleMethod, err)
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
func GetAllMinerDataOnChain(chainModule, chainModuleMethod string) ([]CessChain_AllMinerItems, error) {
	var (
		err   error
		mdata []CessChain_AllMinerItems
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic] [%v.%v] [err:%v]", chainModule, chainModuleMethod, err)
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
			logger.ErrLogger.Sugar().Errorf("[panic] [%v.%v] [err:%v]", chainModule, chainModuleMethod, err)
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
			logger.ErrLogger.Sugar().Errorf("[panic] [%v.%v] [err:%v]", chainModule, chainModuleMethod, err)
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
			logger.ErrLogger.Sugar().Errorf("[panic] [%v.%v] [err:%v]", chainModule, chainModuleMethod, err)
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

//
func (ci *CessInfo) GetDataOnChain() (CessChain_EtcdItems, error) {
	var (
		err   error
		mdata CessChain_EtcdItems
	)

	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic] [%v.%v] [err:%v]", ci.ChainModule, ci.ChainModuleMethod, err)
		}
	}()

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", ci.ChainModule, ci.ChainModuleMethod)
	}

	key, err := types.CreateStorageKey(meta, ci.ChainModule, ci.ChainModuleMethod)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", ci.ChainModule, ci.ChainModuleMethod)
	}

	_, err = api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", ci.ChainModule, ci.ChainModuleMethod)
	}
	return mdata, nil
}

type FileInfo struct {
	FileName       types.Bytes `json:"filename"`
	Owner          types.AccountID
	Filehash       types.Bytes
	Similarityhash types.Bytes
	Ispublic       types.U8
	Backups        types.U8
	Creator        types.Bytes
	Filesize       types.U128
	Keywords       types.Bytes
	Email          types.Bytes
	Uploadfee      types.U128
	Downloadfee    types.U128
	Deadline       types.BlockNumber
}

type MinerInfo struct {
	Address                           types.AccountID
	Beneficiary                       types.AccountID
	Power                             types.U128
	Space                             types.U128
	Total_reward                      types.U128
	Total_rewards_currently_available types.U128
	Totald_not_receive                types.U128
	Collaterals                       types.U128
}

func HexToBytes(s string) []byte {
	s = strings.TrimPrefix(s, "0x")
	c := make([]byte, hex.DecodedLen(len(s)))
	_, _ = hex.Decode(c, []byte(s))
	return c
}

// Query miner information on the cess chain
func GetFileInfoOnChain() {
	// aa, err := signature.KeyringPairFromSecret("leaf obscure fall high office frame make jump nose unusual enrich half", 0)
	// if err != nil {
	// 	panic(err)
	// }
	// Instantiate the API
	api, err := gsrpc.NewSubstrateAPI("ws://106.15.44.155:9947/")
	if err != nil {
		panic(err)
	}
	// key2, err := api.RPC.State.GetStorageRawLatest(types.NewStorageKey(b))
	// if err != nil {
	// 	panic(err)
	// }
	// var da ffff
	// err = json.Unmarshal([]byte(key2.Hex()), &da)
	// if err != nil {
	// 	panic(err)
	// }
	// //types.HexEncodeToString([]byte(key2.Hex()))
	// fmt.Printf("%+v", da)
	// return
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		panic(err)
	}

	// Create a call, transferring 12345 units to Bob
	// bob, err := types.NewMultiAddressFromHexAccountID("0x1e3e1c69dfbd27d398e92da4844a9abdc2786ac01588d87a2e1f5ec06ea2a936")
	// if err != nil {
	// 	panic(err)
	// }

	b := types.MustHexDecodeString("eaf3110750ddb7a01aa8cad56a3dc1f738f306c58ebdaabdbe6623053118986d")
	// if err != nil {
	// 	panic(err)
	// }
	// bbbbb := make([]byte, 0)
	// for _, r := range []rune("eaf3110750ddb7a01aa8cad56a3dc1f738f306c58ebdaabdbe6623053118986d") {
	// 	s := fmt.Sprintf("%c", r)
	// 	in, _ := strconv.Atoi(s)
	// 	bbbbb = append(bbbbb, uint8(in))
	// }
	//types.MustHexDecodeString("eaf3110750ddb7a01aa8cad56a3dc1f738f306c58ebdaabdbe6623053118986d")
	s := fmt.Sprintf("%v", b)
	fmt.Println(s)

	ccc := make([]byte, 0)
	for i := 0; i < len(b); i++ {
		ccc = append(ccc, b...)
	}

	eraIndexSerialized := make([]byte, 8)
	binary.LittleEndian.PutUint64(eraIndexSerialized, uint64(1))

	key, err := types.CreateStorageKey(meta, "FileBank", "File", types.NewBytes([]byte(s)))
	if err != nil {
		panic(err)
	}

	var accountInfo FileInfo
	// b3, err := types.HexDecodeString("0xe75d65720b1ee1c6c10074df3ac67fda0fecdd97bc43e5af5317989742fe4520")
	// if err != nil {
	// 	panic(err)
	// }
	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		fmt.Println("err:", err)
	}

	fmt.Printf("%v\n", accountInfo)
	// ssssss := ""
	// for i := 0; i < len(accountInfo.Address); i++ {
	// 	fmt.Printf("%c ", accountInfo.Address[i])
	// 	temp := fmt.Sprintf("%c", accountInfo.Address[i])
	// 	ssssss += temp
	// }
	// fmt.Println(ssssss)
	//types.NewAccountID(accountInfo.Address)
	// ddd1, err := types.NewAddressFromHexAccountID(ssssss)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("ddd1:%v\n", ddd1)
	// ddd2, err := types.NewMultiAddressFromHexAccountID(ssssss)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("ddd2:%v\n", ddd2)

}
