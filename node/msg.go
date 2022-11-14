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
)

const (
	FileType_Invalid uint8 = iota
	FileType_file
	FileType_filler
)

const (
	Status_Ok Status = iota
	Status_Err
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

func buildNotifyMsg(fileName string, status Status) *Message {
	m := &Message{}
	m.MsgType = MsgNotify
	m.FileName = fileName
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = []byte{byte(status)}
	return m
}

func buildNotifyFillerMsg(fileName string, status Status) *Message {
	m := &Message{}
	m.MsgType = MsgNotify
	m.FileName = ""
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = []byte{byte(status)}
	m.Bytes = append(m.Bytes, []byte(fileName)...)
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

func buildFileMsg(fileName string, filetype uint8, buf []byte) *Message {
	m := &Message{}
	m.MsgType = MsgFile
	m.FileType = filetype
	m.FileName = fileName
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = buf
	return m
}

func buildEndMsg(fileName string, size uint64, lastmark bool) *Message {
	m := &Message{}
	m.MsgType = MsgEnd
	m.FileType = 0
	m.FileName = fileName
	m.FileHash = ""
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
