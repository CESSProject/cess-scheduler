package register

import (
	"fmt"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
)

type CessChain_MinerItems struct {
	Signer     types.AccountID `json:"signer"`
	Ip         types.U32       `json:"ip"`
	Collateral types.U128      `json:"collateral"`
	Earnings   types.U128      `json:"earnings"`
	Locked     types.U128      `json:"locked"`
}

//
func GetChainState(rpc_addr, account_seed, chain_module, module_func string) (err error) {
	// Instantiate the API
	api, err := gsrpc.NewSubstrateAPI(rpc_addr)
	if err != nil {
		return
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return
	}

	account, err := signature.KeyringPairFromSecret(account_seed, 0)
	if err != nil {
		return
	}

	key, err := types.CreateStorageKey(meta, chain_module, module_func, account.PublicKey)
	if err != nil {
		return
	}

	var accountInfo CessChain_MinerItems
	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		fmt.Println(err)
	} else {
		fmt.Println(accountInfo)
	}
	return
}
