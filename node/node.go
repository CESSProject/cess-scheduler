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
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/confile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Scheduler interface {
	Run()
}

type Node struct {
	Confile   confile.Confiler
	Chain     chain.Chainer
	Logs      logger.Logger
	Cache     db.Cacher
	CallBack  *gin.Engine
	FileDir   string
	TagDir    string
	FillerDir string
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
}

func (n *Node) Run() {
	var (
		remote string
	)

	// Start the subtask manager
	go n.CoroutineMgr()

	// Get an address of TCP end point
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", n.Confile.GetServicePort()))
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Listen for TCP networks
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	time.Sleep(time.Second)
	log.Println("Service started successfully")

	for {
		// Connection interval
		time.Sleep(time.Second)

		// Accepts the next connection
		acceptTCP, err := listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println("[err] The port is closed and the service exits.")
				os.Exit(1)
			}
			n.Logs.Common("error", fmt.Errorf("Accept tcp err: %v", err))
			continue
		}

		// Record client address
		remote = acceptTCP.RemoteAddr().String()
		n.Logs.Common("info", fmt.Errorf("Recv a conn: %v", remote))

		// Chain status
		if !n.Chain.GetChainStatus() {
			acceptTCP.Close()
			n.Logs.Common("info", fmt.Errorf("Chain state not available: %v", remote))
			continue
		}

		if !ConnectionFiltering(acceptTCP) {
			acceptTCP.Close()
			n.Logs.Common("info", fmt.Errorf("Close the conn not for a file req: %v ", remote))
			continue
		}

		// Start the processing service of the new connection
		go NewServer(NewTcp(acceptTCP), n.FileDir).Start(n)
	}
}

func ConnectionFiltering(conn *net.TCPConn) bool {
	buf := make([]byte, len(HEAD_FILE))
	_, err := io.ReadAtLeast(conn, buf, len(HEAD_FILE))
	if err != nil {
		return false
	}
	return bytes.Equal(buf, HEAD_FILE)
}
