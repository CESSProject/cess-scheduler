/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

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
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/confile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/CESSProject/cess-scheduler/pkg/serve"
)

type Scheduler interface {
	Run()
}

type Node struct {
	Ser     serve.IServer
	Cfile   confile.IConfile
	Chn     chain.IChain
	Logs    logger.ILog
	Cach    db.ICache
	FileDir string
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
}

func (n *Node) Run() {
	// Start the subtask manager
	go n.CoroutineMgr()

	//
	go serve.AutoExpirationDelete()

	// Start Service
	n.Ser.Serve()
}
