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

// CoroutineMgr is the management program of the cooperation program,
// which can start the cooperation program that unexpectedly exits.
func (node *Node) CoroutineMgr() {
	var (
		ch_common     = make(chan bool, 1)
		ch_minerCache = make(chan bool, 1)
	)

	go node.task_common(ch_common)
	go node.task_minerCache(ch_minerCache)

	for {
		select {
		case <-ch_common:
			go node.task_common(ch_common)
		case <-ch_minerCache:
			go node.task_minerCache(ch_minerCache)
		}
	}
}
