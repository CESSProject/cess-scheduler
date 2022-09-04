package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CESSProject/cess-scheduler/api/protobuf"
	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

func Dial(url string, t time.Duration) error {
	wsURL := "ws://" + string(base58.Decode(url))
	ctx, _ := context.WithTimeout(context.Background(), t)
	cli, err := DialWebsocket(ctx, wsURL, "")
	if err != nil {
		return err
	}
	cli.Close()
	return nil
}

//
func WriteData(dst string, service, method string, t time.Duration, body []byte) ([]byte, error) {
	dstip := "ws://" + string(base58.Decode(dst))
	dstip = strings.Replace(dstip, " ", "", -1)
	req := &protobuf.ReqMsg{
		Service: service,
		Method:  method,
		Body:    body,
	}
	ctx1, _ := context.WithTimeout(context.Background(), 5*time.Second)
	client, err := DialWebsocket(ctx1, dstip, "")
	if err != nil {
		return nil, errors.Wrap(err, "DialWebsocket:")
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()
	resp, err := client.Call(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Call:")
	}

	var b protobuf.RespBody
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal:")
	}
	if b.Code == 200 {
		return b.Data, nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return nil, errors.New("return code:" + errstr)
}
