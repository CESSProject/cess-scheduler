package chain

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/logger"
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

// custom event type
type Event_SegmentBook_ParamSet struct {
	Phase     types.Phase
	PeerId    types.U64
	SegmentId types.U64
	Random    types.U32
	Topics    []types.Hash
}

type Event_VPABCD_Submit_Verify struct {
	Phase     types.Phase
	PeerId    types.U64
	SegmentId types.U64
	Topics    []types.Hash
}

type Event_Sminer_TimedTask struct {
	Phase  types.Phase
	Topics []types.Hash
}

type Event_Sminer_Registered struct {
	Phase   types.Phase
	PeerAcc types.AccountID
	Staking types.U128
	Topics  []types.Hash
}

type Event_UnsignedPhaseStarted struct {
	Phase  types.Phase
	Round  types.U32
	Topics []types.Hash
}

type Event_SolutionStored struct {
	Phase            types.Phase
	Election_compute types.ElectionCompute
	Prev_ejected     types.Bool
	Topics           []types.Hash
}

type Event_FileMap_RegistrationScheduler struct {
	Phase  types.Phase
	Acc    types.AccountID
	Ip     types.Bytes
	Topics []types.Hash
}

type Event_DeleteFile struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_BuySpace struct {
	Phase  types.Phase
	Acc    types.AccountID
	Size   types.U128
	Fee    types.U128
	Topics []types.Hash
}

type Event_FileUpload struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Event_FileUpdate struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

type MyEventRecords struct {
	types.EventRecords
	SegmentBook_ParamSet          []Event_SegmentBook_ParamSet
	SegmentBook_VPASubmitted      []Event_VPABCD_Submit_Verify
	SegmentBook_VPBSubmitted      []Event_VPABCD_Submit_Verify
	SegmentBook_VPCSubmitted      []Event_VPABCD_Submit_Verify
	SegmentBook_VPDSubmitted      []Event_VPABCD_Submit_Verify
	SegmentBook_VPAVerified       []Event_VPABCD_Submit_Verify
	SegmentBook_VPBVerified       []Event_VPABCD_Submit_Verify
	SegmentBook_VPCVerified       []Event_VPABCD_Submit_Verify
	SegmentBook_VPDVerified       []Event_VPABCD_Submit_Verify
	Sminer_TimedTask              []Event_Sminer_TimedTask
	Sminer_Registered             []Event_Sminer_Registered
	FileMap_RegistrationScheduler []Event_FileMap_RegistrationScheduler
	//yz
	FileBank_DeleteFile []Event_DeleteFile
	FileBank_BuySpace   []Event_BuySpace
	FileBank_FileUpload []Event_FileUpload
	FileBank_FileUpdate []Event_FileUpdate
	//
	ElectionProviderMultiPhase_UnsignedPhaseStarted []Event_UnsignedPhaseStarted
	ElectionProviderMultiPhase_SolutionStored       []Event_SolutionStored
}

func RegisterToChain(transactionPrK, TransactionName, ipAddr string) (bool, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
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

	c, err := types.NewCall(meta, TransactionName, types.NewBytes([]byte(ipAddr)))
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

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return false, err
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					fmt.Println("+++ DecodeEvent err: ", err)
				}
				if events.FileMap_RegistrationScheduler != nil {
					for i := 0; i < len(events.FileMap_RegistrationScheduler); i++ {
						if events.FileMap_RegistrationScheduler[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return true, nil
						}
					}
				} else {
					fmt.Println("+++ Not found events.FileMap_RegistrationScheduler")
				}
				return false, errors.New("Not found events.FileMap_RegistrationScheduler")
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

// VerifyInVpaOrVpb
func VerifyInVpaOrVpb(identifyAccountPhrase, TransactionName string, peerid, segid types.U64, result bool) error {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic] [%v] [err:%v]", TransactionName, err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName, peerid, segid, types.NewBool(result))
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx hash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}

// VerifyInVpc
func VerifyInVpc(identifyAccountPhrase, TransactionName string, peerid, segid types.U64, uncid []types.Bytes, result bool) error {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic] [%v] [err:%v]", TransactionName, err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName, peerid, segid, uncid, types.NewBool(result))
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx hash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}

//Submit unsealed cid
func IntentSubmitToChain(identifyAccountPhrase, TransactionName string, segsizetype, segtype uint8, peerid uint64, unsealedcid [][]byte, shardhash []byte) error {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	var uncid []types.Bytes = make([]types.Bytes, len(unsealedcid))
	for i := 0; i < len(unsealedcid); i++ {
		uncid[i] = make(types.Bytes, 0)
		uncid[i] = append(uncid[i], unsealedcid[i]...)
	}
	c, err := types.NewCall(meta, TransactionName, types.NewU8(segsizetype), types.NewU8(segtype), types.NewU64(peerid), uncid, types.NewBytes(shardhash))
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx hash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}

//Submit unsealed cid
func UpdateFileInfoToChain(identifyAccountPhrase, TransactionName string, fid, simhash []byte, ispublic uint8) error {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName, types.NewBytes(fid), types.NewU8(ispublic), types.NewBytes(simhash))
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx blockhash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}

func PutMetaInfoToChain(transactionPrK, TransactionName, fid string, info []FileDuplicateInfo) (bool, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
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

	c, err := types.NewCall(meta, TransactionName, types.Bytes([]byte(fid)), info)
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

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return false, err
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					fmt.Println("+++ DecodeEvent err: ", err)
				}
				if events.FileBank_FileUpdate != nil {
					//fmt.Println("**", events.FileBank_FileUpdate)
					for i := 0; i < len(events.FileBank_FileUpdate); i++ {
						if events.FileBank_FileUpdate[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return true, nil
						}
					}
				} else {
					fmt.Println("+++ Not found events.FileBank_FileUpdate ")
				}
				return false, nil
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

// etcd register
func (ci *CessInfo) RegisterEtcdOnChain(ipAddr string) error {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(ci.IdentifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, ci.TransactionName, types.NewBytes([]byte(ipAddr)))
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx blockhash: %#x", ci.TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", ci.TransactionName)
		default:
			time.Sleep(time.Second)
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
