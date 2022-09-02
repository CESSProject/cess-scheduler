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

package rpc

import (
	"io"

	"github.com/CESSProject/cess-scheduler/api/protobuf"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type protoCodec struct {
	conn     *websocket.Conn
	closedCh chan struct{}
}

func (p *protoCodec) closed() <-chan struct{} {
	return p.closedCh
}

func (p *protoCodec) close() {
	select {
	case <-p.closedCh:
	default:
		close(p.closedCh)
		p.conn.Close()
	}
}

func (p *protoCodec) read(v proto.Message) error {
	_, r, err := p.conn.NextReader()
	if err != nil {
		return err
	}

	bs, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(bs, v)
	return err
}

func (p *protoCodec) write(v proto.Message) error {
	w, err := p.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	bs, _ := proto.Marshal(v)
	_, err = w.Write(bs)
	if err != nil {
		return err
	}

	err = w.Close()
	return err
}

func (p *protoCodec) getConn() *websocket.Conn {
	return p.conn
}

func errorMessage(err error) *protobuf.RespMsg {
	msg := &protobuf.RespMsg{}
	ec, ok := err.(Error)
	errMsg := &protobuf.RespBody{
		Code: defaultErrorCode,
		Msg:  err.Error(),
	}
	if ok {
		errMsg.Code = ec.ErrorCode()
	}
	msg.Body, _ = proto.Marshal(errMsg)
	return msg
}
