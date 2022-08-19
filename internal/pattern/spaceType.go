package pattern

import (
	"cess-scheduler/tools"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type spacemap struct {
	lock   *sync.Mutex
	miners map[string]string
	tokens map[string]authspaceinfo
}

type authspaceinfo struct {
	publicKey  string
	ip         string
	fillerId   string
	updateTime int64
}

var spacem *spacemap

func init() {
	spacem = new(spacemap)
	spacem.lock = new(sync.Mutex)
	spacem.miners = make(map[string]string, 10)
	spacem.tokens = make(map[string]authspaceinfo, 10)
}

func VerifySpaceToken(token string) (string, string, string, error) {
	spacem.lock.Lock()
	v, ok := spacem.tokens[token]
	spacem.lock.Unlock()
	if !ok {
		return "", "", "", errors.New("Invalid token")
	}
	return v.publicKey, v.fillerId, v.ip, nil
}

func UpdateSpacemap(key, ip, fid string) string {
	spacem.lock.Lock()
	defer spacem.lock.Unlock()

	v, _ := spacem.miners[key]
	info, ok2 := spacem.tokens[v]
	if ok2 {
		info.updateTime = time.Now().Unix()
		info.fillerId = fid
		spacem.tokens[v] = info
		return v
	}
	token := tools.RandStr(16)
	data := authspaceinfo{}
	data.publicKey = key
	data.ip = ip
	data.fillerId = fid
	data.updateTime = time.Now().Unix()
	spacem.miners[key] = token
	spacem.tokens[token] = data
	return token
}

func DeleteSpacemap(key string) {
	spacem.lock.Lock()
	v, ok := spacem.tokens[key]
	if ok {
		delete(spacem.miners, v.publicKey)
		delete(spacem.tokens, key)
	}
	spacem.lock.Unlock()
}

func (this *spacemap) DeleteExpired() {
	this.lock.Lock()
	defer this.lock.Unlock()

	for k, v := range this.tokens {
		if time.Since(time.Unix(v.updateTime, 0)).Minutes() > 5 {
			delete(this.miners, v.publicKey)
			delete(this.tokens, k)
		}
	}

	for k, v := range this.miners {
		t, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			if time.Since(time.Unix(t, 0)).Minutes() > 5 {
				delete(this.miners, k)
			}
		}
	}
}

func (this *spacemap) GetConnsMinerNum() int {
	this.lock.Lock()
	defer this.lock.Unlock()
	return len(this.miners)
}

func IsExitSpacem(pubkey string) bool {
	spacem.lock.Lock()
	_, ok := spacem.miners[pubkey]
	spacem.lock.Unlock()
	return ok
}

func (this *spacemap) Connect(pubkey string) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	v, ok := this.miners[pubkey]
	if !ok {
		if len(this.miners) < 50 {
			this.miners[pubkey] = fmt.Sprintf("%v", time.Now().Unix())
			return true
		} else {
			return false
		}
	}

	info, ok := this.tokens[v]
	if ok {
		info.updateTime = time.Now().Unix()
		this.tokens[v] = info
	}
	return true
}
