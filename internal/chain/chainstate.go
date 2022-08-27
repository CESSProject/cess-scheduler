package chain

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

// Get miner information on the chain
func (c *chainClient) GetStorageMinerInfo(pkey []byte) (MinerInfo, error) {
	var data MinerInfo

	if !c.IsChainClientOk() {
		return data, errors.New("rpc connection failed")
	}

	b, err := types.EncodeToBytes(pkey)
	if err != nil {
		return data, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_Sminer,
		Sminer_MinerItems,
		b,
	)
	if err != nil {
		return data, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, errors.New(ERR_Empty)
	}
	return data, nil
}

// Get all miner information on the cess chain
func (c *chainClient) GetAllStorageMiner() ([]types.AccountID, error) {
	var data []types.AccountID

	if !c.IsChainClientOk() {
		return data, errors.New("rpc connection failed")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_Sminer,
		Sminer_AllMinerItems,
	)
	if err != nil {
		return nil, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return nil, errors.New(ERR_Empty)
	}
	return data, nil
}

// Query file meta info
func (c *chainClient) GetFileMetaInfo(fid types.Bytes) (FileMetaInfo, error) {
	var data FileMetaInfo

	if !c.IsChainClientOk() {
		return data, errors.New("rpc connection failed")
	}

	b, err := types.EncodeToBytes(fid)
	if err != nil {
		return data, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_FileBank,
		FileMap_FileMetaInfo,
		b,
	)
	if err != nil {
		return data, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, errors.New(ERR_Empty)
	}
	return data, nil
}

// Query Scheduler info
func (c *chainClient) GetSchedulerInfo() ([]SchedulerInfo, error) {
	var data []SchedulerInfo

	if !c.IsChainClientOk() {
		return data, errors.New("rpc connection failed")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_FileMap,
		FileMap_SchedulerInfo,
	)
	if err != nil {
		return nil, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, errors.New(ERR_Empty)
	}
	return data, nil
}

//
func (c *chainClient) GetProofs() ([]Proof, error) {
	var data []Proof

	if !c.IsChainClientOk() {
		return data, errors.New("rpc connection failed")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_SegmentBook,
		SegmentBook_UnVerifyProof,
		c.keyring.PublicKey,
	)
	if err != nil {
		return nil, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return nil, errors.New(ERR_Empty)
	}
	return data, nil
}

//
func (c *chainClient) GetCessAccount() (types.Bytes, error) {
	acc, err := encodeToCessAccount(c.keyring.PublicKey)
	return types.NewBytes([]byte(acc)), err
}

func encodeToCessAccount(publicKey []byte) (string, error) {
	if len(publicKey) != 32 {
		return "", errors.New("invalid publicKey")
	}
	payload := appendBytes(CessPrefix, publicKey)
	input := appendBytes(SSPrefix, payload)
	ck := blake2b.Sum512(input)
	checkum := ck[:2]
	address := base58.Encode(appendBytes(payload, checkum))
	if address == "" {
		return address, errors.New("base58 encode error")
	}
	return address, nil
}

func appendBytes(data1, data2 []byte) []byte {
	if data2 == nil {
		return data1
	}
	return append(data1, data2...)
}

//
func (c *chainClient) GetSpacePackageInfo() (SpacePackage, error) {
	var data SpacePackage

	if !c.IsChainClientOk() {
		return data, errors.New("rpc connection failed")
	}

	b, err := types.EncodeToBytes(c.keyring.PublicKey)
	if err != nil {
		return data, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_FileBank,
		FileBank_PurchasedPackage,
		b,
	)
	if err != nil {
		return data, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, errors.New(ERR_Empty)
	}
	return data, nil
}

//
func (c *chainClient) GetAccountInfo() (types.AccountInfo, error) {
	var data types.AccountInfo

	if !c.IsChainClientOk() {
		return data, errors.New("rpc connection failed")
	}

	b, err := types.EncodeToBytes(types.NewAccountID(c.keyring.PublicKey))
	if err != nil {
		return data, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(
		c.metadata,
		State_System,
		System_Account,
		b,
	)
	if err != nil {
		return data, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := c.c.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, errors.New(ERR_Empty)
	}
	return data, nil
}
