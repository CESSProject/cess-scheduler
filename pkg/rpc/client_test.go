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
	. "cess-scheduler/api/protobuf"
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

type testService struct{}

func (testService) HelloAction(body []byte) (proto.Message, error) {
	fmt.Println(len(body))
	fmt.Println(string(body[5242870:]))
	return &RespBody{Msg: "test hello"}, nil
}

func TestDialWebsocket(t *testing.T) {
	srv := NewServer()
	srv.Register("test", testService{})
	s := httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
	defer s.Close()
	defer srv.Close()

	wsURL := "ws:" + strings.TrimPrefix(s.URL, "http:")
	fmt.Println(wsURL)
	client, err := DialWebsocket(context.Background(), "ws://127.0.0.1:15000", "")
	if err != nil {
		t.Fatal(err)
	}

	req := &ReqMsg{
		Service: "wservice",
		Method:  "writefile",
	}
	b := make([]byte, 5*1024*1024)
	for i := 0; i < len(b); i++ {
		b[i] = 'a'
	}
	req.Body = b
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := client.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	fmt.Println(resp)
}
