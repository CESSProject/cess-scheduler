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

package task

import (
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/configfile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
)

func Run(
	confile configfile.Configfiler,
	c chain.Chainer,
	db db.Cache,
	logs logger.Logger,
	fillerDir string,
) {
	var (
		channel_1 = make(chan bool, 1)
		channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
		channel_4 = make(chan bool, 1)
		channel_5 = make(chan bool, 1)
	)
	go task_SyncMinersInfo(channel_1, logs, c, db)
	go task_ValidateProof(channel_2, logs, c, db)
	go task_SubmitFillerMeta(channel_3, logs, c, fillerDir)
	go task_GenerateFiller(channel_4, logs, fillerDir)
	go task_ClearAuthMap(channel_5, logs, c)
	for {
		select {
		case <-channel_1:
			go task_SyncMinersInfo(channel_1, logs, c, db)
		case <-channel_2:
			go task_ValidateProof(channel_2, logs, c, db)
		case <-channel_3:
			go task_SubmitFillerMeta(channel_3, logs, c, fillerDir)
		case <-channel_4:
			go task_GenerateFiller(channel_4, logs, fillerDir)
		case <-channel_5:
			go task_ClearAuthMap(channel_5, logs, c)
		}
	}
}
