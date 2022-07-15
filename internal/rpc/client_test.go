package rpc

import (
	. "cess-scheduler/internal/rpc/protobuf"
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
	client, err := DialWebsocket(context.Background(), "ws://113.207.1.32:15000", "")
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
