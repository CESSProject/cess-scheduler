package tools

import (
	"time"

	"github.com/CESSProject/cess-scheduler/pkg/chain"
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
