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
	"sync"
	"sync/atomic"

	"github.com/CESSProject/cess-scheduler/api/protobuf"
)

type ID uint32

type call struct {
	id ID
	ch chan<- protobuf.RespMsg
}

type Client struct {
	conn *ClientConn

	sync.Mutex
	pending   map[ID]call
	id        ID
	closeOnce sync.Once
	closeCh   <-chan struct{}
}

func newClient(codec *websocketCodec) *Client {
	ch := make(chan struct{})
	c := &ClientConn{
		codec:   codec,
		closeCh: ch,
	}
	client := &Client{
		closeCh: ch,
		conn:    c,
		pending: make(map[ID]call),
	}
	client.receive()
	client.dispatch()
	return client
}

func (c *Client) dispatch() {
	go func() {
		for {
			select {
			case <-c.closeCh:
				c.Close()
				return
			}
		}
	}()
}

func (c *Client) receive() {
	go c.conn.readLoop(func(msg protobuf.RespMsg) {
		c.Lock()
		id := ID(msg.Id)
		ca, exist := c.pending[id]
		if exist {
			delete(c.pending, id)
		}
		c.Unlock()

		if exist {
			ca.ch <- msg
		}
	})
}

func (c *Client) nextId() ID {
	n := atomic.AddUint32((*uint32)(&c.id), 1)
	return ID(n)
}

func (c *Client) Call(ctx context.Context, msg *protobuf.ReqMsg) (*protobuf.RespMsg, error) {
	ch := make(chan protobuf.RespMsg)
	ca := call{
		id: c.nextId(),
		ch: ch,
	}
	msg.Id = uint64(ca.id)

	c.Lock()
	c.pending[ca.id] = ca
	c.Unlock()

	err := c.conn.codec.WriteMsg(ctx, msg)
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return &resp, nil
	case <-ctx.Done():
		c.Lock()
		delete(c.pending, ca.id)
		c.Unlock()
		return nil, ctx.Err()
	}
}

func (c *Client) Close() {
	c.conn.codec.close()
	c.closeOnce.Do(func() {
		c.Lock()
		defer c.Unlock()
		for id, ca := range c.pending {
			close(ca.ch)
			delete(c.pending, id)
		}
	})
}
