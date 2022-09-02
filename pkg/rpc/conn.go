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
	"context"

	"github.com/CESSProject/cess-scheduler/api/protobuf"
	"github.com/golang/protobuf/proto"
)

type SrvConn struct {
	srv   *Server
	codec *websocketCodec
}

func (c *SrvConn) readLoop() {
	defer c.codec.close()
	for {
		msg := protobuf.ReqMsg{}
		err := c.codec.read(&msg)
		if _, ok := err.(*proto.ParseError); ok {
			c.codec.WriteMsg(context.Background(), errorMessage(&parseError{err.Error()}))
			continue
		}

		if err != nil {
			//log.Debug("server RPC connection read error ", err)
			c.codec.Close()
			break
		}

		c.srv.handle(&msg, c)
	}
}

type ClientConn struct {
	codec   *websocketCodec
	closeCh chan<- struct{}
}

func (c *ClientConn) readLoop(recv func(msg protobuf.RespMsg)) {
	for {
		msg := protobuf.RespMsg{}
		err := c.codec.read(&msg)
		if _, ok := err.(*proto.ParseError); ok {
			continue
		}

		if err != nil {
			//log.Debug("client RPC connection read error ", err)
			c.closeCh <- struct{}{}
			break
		}
		recv(msg)
	}
}
