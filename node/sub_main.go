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

// CoroutineMgr is the management program of the cooperation program,
// which can start the cooperation program that unexpectedly exits.
func (node *Node) CoroutineMgr() {
	var (
		channel_1 = make(chan bool, 1)
		channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
		channel_4 = make(chan bool, 1)
		channel_5 = make(chan bool, 1)
		channel_6 = make(chan bool, 1)
	)

	go node.task_MinerCache(channel_1)
	go node.task_ValidateProof(channel_2)
	go node.task_SubmitFillerMeta(channel_3)
	go node.task_GenerateFiller(channel_4)
	go node.task_Common(channel_5)
	go node.task_Space(channel_6)

	for {
		select {
		case <-channel_1:
			go node.task_MinerCache(channel_1)
		case <-channel_2:
			go node.task_ValidateProof(channel_2)
		case <-channel_3:
			go node.task_SubmitFillerMeta(channel_3)
		case <-channel_4:
			go node.task_GenerateFiller(channel_4)
		case <-channel_5:
			go node.task_Common(channel_5)
		case <-channel_6:
			go node.task_Space(channel_6)
		}
	}
}
