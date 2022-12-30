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

package serve

import (
	"sync"
	"time"
)

type blacklistMiner struct {
	Lock *sync.Mutex
	List map[string]int64
}

var BlackMiners *blacklistMiner

func init() {
	BlackMiners = &blacklistMiner{
		Lock: new(sync.Mutex),
		List: make(map[string]int64, 100),
	}
}

func (b *blacklistMiner) Add(ip string) {
	b.Lock.Lock()
	b.List[ip] = time.Now().Unix()
	b.Lock.Unlock()
}

func (b *blacklistMiner) Delete(ip string) {
	b.Lock.Lock()
	delete(b.List, ip)
	b.Lock.Unlock()
}

func (b *blacklistMiner) IsExist(ip string) bool {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	v, ok := b.List[ip]
	if !ok {
		return false
	}
	if time.Since(time.Unix(v, 0)).Hours() > 3 {
		delete(b.List, ip)
		return false
	}
	return true
}
