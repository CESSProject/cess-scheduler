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

package node

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"sync"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
)

type TcpCon struct {
	conn *net.TCPConn

	recv chan *Message
	send chan *Message

	onceStop *sync.Once
	stop     chan struct{}
}

var (
	HEAD_FILE   = []byte("c100")
	HEAD_FILLER = []byte("c101")
)

func NewTcp(conn *net.TCPConn) *TcpCon {
	return &TcpCon{
		conn:     conn,
		recv:     make(chan *Message, configs.TCP_Message_Read_Buffers),
		send:     make(chan *Message, configs.TCP_Message_Send_Buffers),
		onceStop: new(sync.Once),
		stop:     make(chan struct{}),
	}
}

func (t *TcpCon) HandlerLoop(wg *sync.WaitGroup, flag bool) {
	go t.readMsg(wg, flag)
	go t.sendMsg(wg)
}

func (t *TcpCon) sendMsg(wg *sync.WaitGroup) {
	sendBuf := readBufPool.Get().([]byte)
	wg.Add(1)
	defer func() {
		wg.Done()
		recover()
		t.Close()
		time.Sleep(time.Second)
		close(t.send)
		readBufPool.Put(sendBuf)
	}()

	copy(sendBuf[:len(HEAD_FILLER)], HEAD_FILLER)

	for !t.IsClose() {
		select {
		case m := <-t.send:
			data, err := json.Marshal(m)
			if err != nil {
				return
			}

			switch cap(m.Bytes) {
			case configs.TCP_TagBuffer:
				tagBufPool.Put(m.Bytes)
			case configs.TCP_SendBuffer:
				sendBufPool.Put(m.Bytes)
			default:
			}

			binary.BigEndian.PutUint32(sendBuf[len(HEAD_FILLER):len(HEAD_FILLER)+4], uint32(len(data)))
			copy(sendBuf[len(HEAD_FILLER)+4:], data)

			_, err = t.conn.Write(sendBuf[:len(HEAD_FILLER)+4+len(data)])
			if err != nil {
				return
			}
		default:
			time.Sleep(configs.TCP_Message_Interval)
		}
	}
}

func (t *TcpCon) readMsg(wg *sync.WaitGroup, flag bool) {
	var (
		err      error
		n        int
		waittime int64
		header   = make([]byte, 4)
	)
	wg.Add(1)
	readBuf := readBufPool.Get().([]byte)
	defer func() {
		wg.Done()
		recover()
		t.Close()
		close(t.recv)
		readBufPool.Put(readBuf)
	}()

	for !t.IsClose() {
		if !flag {
			// read until we get 4 bytes for the magic
			_, err = io.ReadAtLeast(t.conn, header, 4)
			if err != nil {
				if err != io.EOF {
					return
				}

				if err == io.EOF {
					waittime++
					if waittime >= 10 {
						return
					}
					time.Sleep(time.Second)
					continue
				}
			}

			if !bytes.Equal(header, HEAD_FILLER) && !bytes.Equal(header, HEAD_FILE) {
				return
			}
		}
		flag = false

		// read until we get 4 bytes for the header
		_, err = io.ReadAtLeast(t.conn, header, 4)
		if err != nil {
			return
		}

		// data size
		msgSize := binary.BigEndian.Uint32(header)

		// read data
		if msgSize > configs.TCP_ReadBuffer {
			return
		}

		n, err = io.ReadFull(t.conn, readBuf[:msgSize])
		if err != nil {
			return
		}
		m := &Message{}
		m.Bytes = readBufPool.Get().([]byte)

		err = json.Unmarshal(readBuf[:n], &m)
		if err != nil {
			return
		}

		t.recv <- m
	}
}

func (t *TcpCon) GetMsg() (*Message, bool) {
	timer := time.NewTimer(configs.TCP_Time_WaitMsg)
	defer timer.Stop()
	select {
	case m, ok := <-t.recv:
		return m, ok
	case <-timer.C:
		return nil, true
	}
}

func (t *TcpCon) SendMsg(m *Message) {
	t.send <- m
}

func (t *TcpCon) GetRemoteAddr() string {
	return t.conn.RemoteAddr().String()
}

func (t *TcpCon) Close() error {
	t.onceStop.Do(func() {
		time.Sleep(time.Second * 3)
		t.conn.Close()
		close(t.stop)
	})
	return nil
}

func (t *TcpCon) IsClose() bool {
	select {
	case <-t.stop:
		return true
	default:
		return false
	}
}

var _ = NetConn(&TcpCon{})
