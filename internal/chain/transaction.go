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

	c, err := types.NewCall(meta, TransactionName, types.Bytes([]byte(ipAddr)))
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

// VerifyInVpaOrVpb
func VerifyInVpaOrVpbOrVpd(identifyAccountPhrase, TransactionName string, peerid, segid types.U64, result bool) error {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v] [err:%v]", TransactionName, err)
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

	c, err := types.NewCall(meta, TransactionName, peerid, segid, types.Bool(result))
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

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey System Events err")
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
						return errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return err
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
				switch TransactionName {
				case configs.ChainTx_SegmentBook_VerifyInVpa:
					if events.SegmentBook_VPAVerified != nil {
						for i := 0; i < len(events.SegmentBook_VPAVerified); i++ {
							if events.SegmentBook_VPAVerified[i].PeerId == peerid && events.SegmentBook_VPAVerified[i].SegmentId == segid {
								return nil
							}
						}
						if head != nil {
							return errors.Errorf("[%v]events.SegmentBook_VPAVerified data err", head.Number)
						} else {
							return errors.New("events.SegmentBook_VPAVerified data err")
						}
					}
					if head != nil {
						return errors.Errorf("[%v]events.SegmentBook_VPAVerified not found", head.Number)
					} else {
						return errors.New("events.SegmentBook_VPAVerified not found")
					}
				case configs.ChainTx_SegmentBook_VerifyInVpb:
					if events.SegmentBook_VPBVerified != nil {
						for i := 0; i < len(events.SegmentBook_VPBVerified); i++ {
							if events.SegmentBook_VPBVerified[i].PeerId == peerid && events.SegmentBook_VPBVerified[i].SegmentId == segid {
								return nil
							}
						}
						if head != nil {
							return errors.Errorf("[%v]events.SegmentBook_VPBVerified data err", head.Number)
						} else {
							return errors.New("events.SegmentBook_VPBVerified data err")
						}
					}
					if head != nil {
						return errors.Errorf("[%v]events.SegmentBook_VPBVerified not found", head.Number)
					} else {
						return errors.New("events.SegmentBook_VPBVerified not found")
					}
				case configs.ChainTx_SegmentBook_VerifyInVpd:
					if events.SegmentBook_VPDVerified != nil {
						for i := 0; i < len(events.SegmentBook_VPDVerified); i++ {
							if events.SegmentBook_VPDVerified[i].PeerId == peerid && events.SegmentBook_VPDVerified[i].SegmentId == segid {
								return nil
							}
						}
						if head != nil {
							return errors.Errorf("[%v]events.SegmentBook_VPDVerified data err", head.Number)
						} else {
							return errors.New("events.SegmentBook_VPDVerified data err")
						}
					}
					if head != nil {
						return errors.Errorf("[%v]events.SegmentBook_VPDVerified not found", head.Number)
					} else {
						return errors.New("events.SegmentBook_VPDVerified not found")
					}
				}
				if head != nil {
					return errors.Errorf("[%v]events.SegmentBook_VPA_or_VPB_Verified not found", head.Number)
				} else {
					return errors.New("events.SegmentBook_VPA_or_VPB_Verified not found")
				}
			}
		case err = <-sub.Err():
			return err
		case <-timeout:
			return errors.Errorf("[%v][%v][%v] tx timeout", peerid, segid, TransactionName)
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
			Err.Sugar().Errorf("[panic] [%v][%v][%v] [err:%v]", peerid, segid, TransactionName, err)
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

	c, err := types.NewCall(meta, TransactionName, peerid, segid, uncid, types.Bool(result))
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

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey System Events err")
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
						return errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return err
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
				if events.SegmentBook_VPCVerified != nil {
					for i := 0; i < len(events.SegmentBook_VPCVerified); i++ {
						if events.SegmentBook_VPCVerified[i].PeerId == peerid && events.SegmentBook_VPCVerified[i].SegmentId == segid {
							return nil
						}
					}
					if head != nil {
						return errors.Errorf("[%v]events.SegmentBook_VPCVerified data err", head.Number)
					} else {
						return errors.New("events.SegmentBook_VPCVerified data err")
					}
				}
				if head != nil {
					return errors.Errorf("[%v]events.SegmentBook_VPCVerified not found", head.Number)
				} else {
					return errors.New("events.SegmentBook_VPCVerified not found")
				}
			}
		case err = <-sub.Err():
			return err
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
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
			Err.Sugar().Errorf("[panic]: %v", err)
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
	c, err := types.NewCall(meta, TransactionName, types.U8(segsizetype), types.U8(segtype), types.U64(peerid), uncid, types.NewBytes(shardhash))
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

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey System Events err")
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
						return errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return err
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
				if events.SegmentBook_ParamSet != nil {
					for i := 0; i < len(events.SegmentBook_ParamSet); i++ {
						if events.SegmentBook_ParamSet[i].PeerId == types.U64(peerid) {
							return nil
						}
					}
					if head != nil {
						return errors.Errorf("[%v]events.SegmentBook_ParamSet data err", head.Number)
					} else {
						return errors.New("events.SegmentBook_ParamSet data err")
					}
				}
				if head != nil {
					return errors.Errorf("[%v]events.SegmentBook_ParamSet not found", head.Number)
				} else {
					return errors.New("events.SegmentBook_ParamSet not found")
				}
			}
		case err = <-sub.Err():
			return err
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		}
	}
}

// Update file meta information
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
			return false, errors.Errorf("[%v] tx timeout", TransactionName)
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
