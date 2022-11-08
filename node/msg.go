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
	"encoding/json"
	"sync"
)

type MsgType byte

const (
	MsgInvalid MsgType = iota
	MsgHead
	MsgFile
	MsgEnd
	MsgNotify
	MsgClose
	MsgRecvHead
	MsgRecvFile
	MsgFillerHead
	MsgFiller
	MsgFillerEnd
)

type Status byte

const (
	Status_Ok Status = iota
	Status_Err
)

type Message struct {
	MsgType  MsgType `json:"msg_type"`
	FileName string  `json:"file_name"`
	FileHash string  `json:"file_hash"`
	FileSize uint64  `json:"file_size"`
	LastMark bool    `json:"last_mark"`
	Pubkey   []byte  `json:"pub_key"`
	SignMsg  []byte  `json:"sign_msg"`
	Sign     []byte  `json:"sign"`
	Bytes    []byte  `json:"bytes"`
}

type Notify struct {
	Status byte
}

var (
	msgPool = &sync.Pool{
		New: func() interface{} {
			return &Message{}
		},
	}

	bytesPool = &sync.Pool{
		New: func() interface{} {
			mem := make([]byte, 40*1024)
			return &mem
		},
	}
)

func (m *Message) GC() {
	if m != nil {
		m.reset()
		msgPool.Put(m)
	}
}

func (m *Message) reset() {
	if m != nil {
		m.MsgType = MsgInvalid
		m.FileName = ""
		m.FileHash = ""
		m.FileSize = 0
		m.LastMark = false
		m.Pubkey = nil
		m.SignMsg = nil
		m.Sign = nil
		m.Bytes = nil
	} else {
		m = &Message{}
	}
}

func (m *Message) String() (string, error) {
	bytes, err := json.Marshal(m)
	return string(bytes), err
}

// Decode will convert from bytes
func Decode(b []byte) (m *Message, err error) {
	m = msgPool.Get().(*Message)
	err = json.Unmarshal(b, &m)
	return
}

func (m *Message) buildNotifyMsg(fileName string, status Status) {
	m.MsgType = MsgNotify
	m.Bytes = []byte{byte(status)}
	m.FileName = fileName
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
}

func NewNotifyFillerMsg(fileName string, status Status) *Message {
	m := msgPool.Get().(*Message)
	m.reset()
	m.MsgType = MsgNotify
	m.Bytes = []byte{byte(status)}
	m.Bytes = append(m.Bytes, []byte(fileName)...)
	return m
}

func NewHeadMsg(fileName string, fid string, lastmark bool, pkey, signmsg, sign []byte) *Message {
	m := msgPool.Get().(*Message)
	m.reset()
	m.MsgType = MsgHead
	m.FileName = fileName
	m.FileHash = fid
	m.LastMark = lastmark
	m.Pubkey = pkey
	m.SignMsg = signmsg
	m.Sign = sign
	return m
}

func NewFileMsg(fileName string, buf []byte) *Message {
	m := msgPool.Get().(*Message)
	m.reset()
	m.MsgType = MsgFile
	m.FileName = fileName
	m.Bytes = buf
	return m
}

func (m *Message) buildFillerMsg(fileName string, buf []byte) {
	m.MsgType = MsgFiller
	m.FileName = fileName
	m.FileHash = fileName
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = buf
}

func NewEndMsg(fileName string, size uint64, lastmark bool) *Message {
	m := msgPool.Get().(*Message)
	m.reset()
	m.MsgType = MsgEnd
	m.FileName = fileName
	m.FileSize = size
	m.LastMark = lastmark
	return m
}

func (m *Message) buildFillerEndMsg(fileName string, size uint64) {
	m.MsgType = MsgFillerEnd
	m.FileName = fileName
	m.FileHash = fileName
	m.FileSize = size
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = nil
}

func (m *Message) buildCloseMsg(fileName string, status Status) {
	m.MsgType = MsgClose
	m.FileName = fileName
	m.FileHash = fileName
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = []byte{byte(status)}
}
