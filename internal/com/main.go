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
	"log"
	"net/http"
	"os"

	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/configfile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/CESSProject/cess-scheduler/pkg/rpc"
)

type WService struct {
	configfile.Configfiler
	logger.Logger
	db.Cache
	chain.Chainer
	fillerDir string
	fileDir   string
}

// Start tcp service.
// If an error occurs, it will exit immediately.
func Start(
	cfg configfile.Configfiler,
	cli chain.Chainer,
	db db.Cache,
	logs logger.Logger,
	fillerDir string,
	fileDir string,
) {
	srv := rpc.NewServer()
	err := srv.Register(
		RpcService_Scheduler,
		&WService{cfg, logs, db, cli, fillerDir, fileDir},
	)
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}
	log.Println("Start and listen on port ", cfg.GetServicePort(), "...")
	err = http.ListenAndServe(
		":"+cfg.GetServicePort(),
		srv.WebsocketHandler([]string{"*"}),
	)
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}
}
