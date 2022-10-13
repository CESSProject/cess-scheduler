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

package com

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/CESSProject/cess-scheduler/internal/tools"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
)

const (
	MAX_TCP_CONNECTION uint32 = 3
)

var TCP_ConnLength atomic.Uint32

type Server interface {
	Start(cli chain.Chainer)
}

type Client interface {
	SendFile() error
}

type ConMgr struct {
	conn       NetConn
	dir        string
	fileName   string
	sendFiles  []string
	waitNotify chan bool
	stop       chan struct{}
}

func init() {
	TCP_ConnLength.Store(0)
}

func NewServer(conn NetConn, dir string) Server {
	return &ConMgr{
		conn: conn,
		dir:  dir,
		stop: make(chan struct{}),
	}
}

func (c *ConMgr) Start(cli chain.Chainer) {
	c.conn.HandlerLoop()
	err := c.handler(cli)
	if err != nil {
		log.Println(err)
	}
}

func (c *ConMgr) handler(cli chain.Chainer) error {
	var fs *os.File
	var err error
	num := TCP_ConnLength.Load()
	num += 1
	TCP_ConnLength.Store(num)
	defer func() {
		if fs != nil {
			_ = fs.Close()
		}
		num := TCP_ConnLength.Load()
		if num > 0 {
			num -= 1
		}
		TCP_ConnLength.Store(num)
	}()

	for !c.conn.IsClose() {
		m, ok := c.conn.GetMsg()
		if !ok {
			return fmt.Errorf("close by connect")
		}
		if m == nil {
			continue
		}

		switch m.MsgType {
		case MsgHead:
			// Verify signature
			ok, err := tools.VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil {
				fmt.Println("VerifySign err: ", err)
				c.conn.SendMsg(NewNotifyMsg(c.fileName, Status_Err))
				return err
			}
			if !ok {
				fmt.Println("Verify Sign failed: ", err)
				c.conn.SendMsg(NewNotifyMsg(c.fileName, Status_Err))
				return err
			}

			//Judge whether the file has been uploaded
			fileState, err := tools.GetFileState(cli, m.FileHash)
			if err != nil {
				fmt.Println("GetFileState err: ", err)
				c.conn.SendMsg(NewNotifyMsg(c.fileName, Status_Err))
				return err
			}
			if fileState == chain.FILE_STATE_ACTIVE {
				fmt.Println("Repeated upload: ", m.FileHash)
				c.conn.SendMsg(NewNotifyMsg(c.fileName, Status_Err))
				return err
			}

			//TODO: Judge whether the space is enough

			if m.FileName != "" {
				c.fileName = m.FileHash
			} else {
				c.fileName = m.FileHash
			}

			fmt.Println("recv head fileName is", c.fileName)
			fs, err = os.OpenFile(filepath.Join(c.dir, c.fileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				fmt.Println("os.Create err =", err)
				c.conn.SendMsg(NewNotifyMsg(c.fileName, Status_Err))
				return err
			}
			fmt.Println("send head is ok")

			c.conn.SendMsg(NewNotifyMsg(c.fileName, Status_Ok))
		case MsgFile:
			if fs == nil {
				fmt.Println(c.fileName, "file is not open !")
				c.conn.SendMsg(NewCloseMsg(c.fileName, Status_Err))
				return nil
			}
			_, err = fs.Write(m.Bytes)
			if err != nil {
				fmt.Println("file.Write err =", err)
				c.conn.SendMsg(NewCloseMsg(c.fileName, Status_Err))
				return err
			}
		case MsgEnd:
			info, _ := fs.Stat()
			if info.Size() != int64(m.FileSize) {
				err = fmt.Errorf("file.size %v rece size %v \n", info.Size(), m.FileSize)
				c.conn.SendMsg(NewCloseMsg(c.fileName, Status_Err))
				return err
			}

			fmt.Printf("save file %v is success \n", info.Name())
			c.conn.SendMsg(NewNotifyMsg(c.fileName, Status_Ok))

			fmt.Printf("close file %v is success \n", c.fileName)
			_ = fs.Close()
			fs = nil
			// TODO: save file to miner
		case MsgNotify:
			c.waitNotify <- m.Bytes[0] == byte(Status_Ok)
		case MsgClose:
			fmt.Printf("revc close msg ....\n")
			if m.Bytes[0] != byte(Status_Ok) {
				return fmt.Errorf("server an error occurred")
			}
			return nil
		}
	}

	return err
}
