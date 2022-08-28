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

	. "cess-scheduler/api/protobuf"

	"github.com/golang/protobuf/proto"
)

type handleWrapper func(id uint64, body []byte) *RespMsg
type Handler func(body []byte) (proto.Message, error)

func (s *Server) handle(msg *ReqMsg, c *SrvConn) {
	s.processMsg(func(ctx context.Context) {
		answer := s.callMethod(msg)
		c.codec.WriteMsg(ctx, answer)
	})
}

func (s *Server) processMsg(fn func(ctx context.Context)) {
	s.handleWg.Add(1)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer s.handleWg.Done()
		defer cancel()
		fn(ctx)
	}()
}

func (s *Server) callMethod(msg *ReqMsg) *RespMsg {
	handler := s.router.lookup(msg.Service, msg.Method)
	if handler == nil {
		err := &methodNotFoundError{msg.Service + "." + msg.Method}
		answer := errorMessage(err)
		answer.Id = msg.Id
		return answer
	}
	answer := handler(msg.Id, msg.Body)
	return answer
}
