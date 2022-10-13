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
	"errors"
	"log"
	"net"
	"os"
	"time"

	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/configfile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
)

// type WService struct {
// 	configfile.Configfiler
// 	logger.Logger
// 	db.Cache
// 	chain.Chainer
// 	fillerDir string
// 	fileDir   string
// }

// Start is used to start the tcp service.
func Start(
	cfg configfile.Configfiler,
	cli chain.Chainer,
	db db.Cache,
	logs logger.Logger,
	fillerDir string,
	fileDir string,
) {
	// Get an address of TCP end point
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+cfg.GetServicePort())
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

	for {
		// Accepts the next connection
		acceptTCP, err := listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println(err)
				os.Exit(1)
			}
			log.Println("Accept tcp err: ", err)
			continue
		}

		log.Println("Get conn remote addr: ", acceptTCP.RemoteAddr().String())

		// Set server maximum connection control
		if TCP_ConnLength.Load() > MAX_TCP_CONNECTION {
			acceptTCP.Close()
			log.Println("Connecion is max, close conn: ", acceptTCP.RemoteAddr().String())
			continue
		}

		// Start the processing service of the new connection
		tcpCon := NewTcp(acceptTCP)
		srv := NewServer(tcpCon, fileDir)
		go srv.Start(cli)
		time.Sleep(time.Millisecond)
	}
}
