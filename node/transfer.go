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
	"fmt"
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
	MAGIC_BYTES = []byte("cess")
	EmErr       = fmt.Errorf("dont have msg")
)

func NewTcp(conn *net.TCPConn) *TcpCon {
	return &TcpCon{
		conn:     conn,
		recv:     make(chan *Message, configs.TCP_Message_Buffers),
		send:     make(chan *Message, configs.TCP_Message_Buffers),
		onceStop: &sync.Once{},
		stop:     make(chan struct{}),
	}
}

func (t *TcpCon) HandlerLoop() {
	go t.readMsg()
	go t.sendMsg()
}

func (t *TcpCon) sendMsg() {
	var (
		err error
		buf = make([]byte, configs.TCP_Write_Buf)
	)
	defer func() {
		_ = t.Close()
	}()

	for !t.IsClose() {
		select {
		case m := <-t.send:
			data := m.String()
			m.GC()

			dataLen := len(data)

			copy(buf[:4], MAGIC_BYTES)
			binary.BigEndian.PutUint32(buf[4:8], uint32(dataLen))
			copy(buf[8:], []byte(data))

			_, err = t.conn.Write(buf[:8+dataLen])
			if err != nil {
				return
			}
		}
	}
}

func (t *TcpCon) readMsg() {
	var (
		err    error
		header = make([]byte, 4)
		buf    = make([]byte, configs.TCP_Read_Buf)
	)
	defer func() {
		_ = t.Close()
	}()

	for {
		// read until we get 4 bytes for the magic
		_, err = io.ReadFull(t.conn, header)
		if err != nil {
			if err != io.EOF {
				err = fmt.Errorf("initial read error: %v \n", err)
				return
			}
			time.Sleep(time.Millisecond)
			continue
		}

		if !bytes.Equal(header, MAGIC_BYTES) {
			err = fmt.Errorf("initial bytes are not magic: %s", header)
			return
		}

		// read until we get 4 bytes for the header
		_, err = io.ReadFull(t.conn, header)
		if err != nil {
			err = fmt.Errorf("initial read error: %v \n", err)
			return
		}

		// data size
		msgSize := binary.BigEndian.Uint32(header)

		var n int
		var m *Message

		n, err = io.ReadFull(t.conn, buf[:msgSize])
		if err != nil {
			err = fmt.Errorf("initial read error: %v \n", err)
			return
		}

		m, err = Decode(buf[:n])
		if err != nil {
			err = fmt.Errorf("read message error: %v \n", err)
			return
		}

		t.recv <- m
	}
}

func (t *TcpCon) GetMsg() (*Message, bool) {
	timer := time.NewTimer(5 * time.Second)
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
		fmt.Println("close a connect, addr: ", t.conn.RemoteAddr())
		_ = t.conn.Close()
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
