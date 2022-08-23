package rpc

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"storj.io/common/base58"
)

func Dial(url string) error {
	wsURL := "ws://" + string(base58.Decode(url))
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	cli, err := rpc.DialWebsocket(ctx, wsURL, "")
	if err != nil {
		return err
	}
	cli.Close()
	return nil
}
