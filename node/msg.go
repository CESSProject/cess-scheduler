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
	"sync"

	"github.com/CESSProject/cess-scheduler/configs"
)

type MsgType byte
type Status byte

const (
	MsgInvalid MsgType = iota
	MsgHead
	MsgFile
	MsgEnd
	MsgNotify
	MsgClose
	MsgRecvHead
	MsgRecvFile
	MsgFileSt
	MsgVersion
)

const (
	FileType_Invalid uint8 = iota
	FileType_file
	FileType_filler
)

const (
	Status_Ok Status = iota
	Status_Err
	Status_Exists
)

var (
	sendBufPool = &sync.Pool{
		New: func() any {
			return make([]byte, configs.TCP_SendBuffer)
		},
	}

	readBufPool = &sync.Pool{
		New: func() any {
			return make([]byte, configs.TCP_ReadBuffer)
		},
	}

	tagBufPool = &sync.Pool{
		New: func() any {
			return make([]byte, configs.TCP_TagBuffer)
		},
	}
)

type Message struct {
	Pubkey   []byte  `json:"pubkey"`
	SignMsg  []byte  `json:"signmsg"`
	Sign     []byte  `json:"sign"`
	Bytes    []byte  `json:"bytes"`
	FileName string  `json:"filename"`
	FileHash string  `json:"filehash"`
	FileSize uint64  `json:"filesize"`
	MsgType  MsgType `json:"msgtype"`
	LastMark bool    `json:"lastmark"`
	FileType uint8   `json:"filetype"`
}

type Notify struct {
	Status byte
}

func buildNotifyMsg(fileName string, status Status, ver string) *Message {
	m := &Message{}
	m.MsgType = MsgNotify
	if fileName != "" {
		m.FileName = fileName
	} else {
		m.FileName = ver
	}
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = []byte{byte(status)}
	return m
}

func buildHeadMsg(filename, fid string, filetype uint8, lastmark bool, pkey, signmsg, sign []byte) *Message {
	m := &Message{}
	m.MsgType = MsgHead
	m.FileType = filetype
	m.FileName = filename
	m.FileHash = fid
	m.FileSize = 0
	m.LastMark = lastmark
	m.Pubkey = pkey
	m.SignMsg = signmsg
	m.Sign = sign
	m.Bytes = nil
	return m
}

func buildFileMsg(fileName string, filetype uint8, buflen int, buf []byte) *Message {
	m := &Message{}
	m.MsgType = MsgFile
	m.FileType = filetype
	m.FileName = fileName
	m.FileHash = ""
	m.FileSize = uint64(buflen)
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	if filetype == FileType_filler && buflen == configs.TCP_TagBuffer {
		m.Bytes = tagBufPool.Get().([]byte)
	} else {
		m.Bytes = sendBufPool.Get().([]byte)
	}
	copy(m.Bytes, buf)
	return m
}

func buildEndMsg(filetype uint8, fileName, fileHash string, size uint64, lastmark bool) *Message {
	m := &Message{}
	m.MsgType = MsgEnd
	m.FileType = filetype
	m.FileName = fileName
	m.FileHash = fileHash
	m.FileSize = size
	m.LastMark = lastmark
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = nil
	return m
}

func buildCloseMsg(status Status) *Message {
	m := &Message{}
	m.MsgType = MsgClose
	m.FileType = 0
	m.FileName = ""
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = []byte{byte(status)}
	return m
}

func buildFileStMsg(fid string, val []byte) *Message {
	m := &Message{}
	m.MsgType = MsgFileSt
	m.FileType = 0
	m.FileName = ""
	m.FileHash = fid
	m.FileSize = uint64(len(val))
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = val
	return m
}

func buildVersionMsg(ver string) *Message {
	m := &Message{}
	m.MsgType = MsgVersion
	m.FileName = ver
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = []byte{byte(Status_Ok)}
	return m
}
