/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package chain

import (
	"cess-scheduler/pkg/utils"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// GetPublicKey returns your own public key
func (c *chainClient) GetPublicKey() []byte {
	return c.keyring.PublicKey
}

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
func (c *chainClient) GetCessAccount() (string, error) {
	return utils.EncodePublicKeyAsCessAccount(c.keyring.PublicKey)
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
