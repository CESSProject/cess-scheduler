package node

import (
	"errors"
	"time"

	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func GetFileState(c chain.Chainer, fileHash string) (string, error) {
	var try_count uint8
	for {
		fmeta, err := c.GetFileMetaInfo(types.NewBytes([]byte(fileHash)))
		if err != nil {
			try_count++
			if try_count > 3 {
				return "", err
			}
			time.Sleep(time.Second * time.Duration(try_count))
			continue
		}
		return string(fmeta.State), nil
	}
}

func VerifySign(pkey, signmsg, sign []byte) (bool, error) {
	if len(signmsg) == 0 || len(sign) < 64 {
		return false, errors.New("Wrong signature")
	}

	ss58, err := utils.EncodePublicKeyAsSubstrateAccount(pkey)
	if err != nil {
		return false, err
	}

	verkr, _ := keyring.FromURI(ss58, keyring.NetSubstrate{})

	var sign_array [64]byte
	for i := 0; i < 64; i++ {
		sign_array[i] = sign[i]
	}

	// Verify signature
	return verkr.Verify(verkr.SigningContext(signmsg), sign_array), nil
}
