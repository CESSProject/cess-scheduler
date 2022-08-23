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

var sm *spacemap

func init() {
	sm = new(spacemap)
	sm.lock = new(sync.Mutex)
	sm.miners = make(map[string]string, 10)
	sm.tokens = make(map[string]authspaceinfo, 10)
}

func VerifySpaceToken(token string) (string, string, string, error) {
	sm.lock.Lock()
	v, ok := sm.tokens[token]
	sm.lock.Unlock()
	if !ok {
		return "", "", "", errors.New("Invalid token")
	}
	return v.publicKey, v.fillerId, v.ip, nil
}

func (this *spacemap) UpdateSpacemap(key, ip, fid string) string {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, _ := this.miners[key]
	info, ok2 := this.tokens[v]
	if ok2 {
		info.updateTime = time.Now().Unix()
		info.fillerId = fid
		this.tokens[v] = info
		return v
	}
	token := tools.RandStr(16)
	data := authspaceinfo{}
	data.publicKey = key
	data.ip = ip
	data.fillerId = fid
	data.updateTime = time.Now().Unix()
	this.miners[key] = token
	this.tokens[token] = data
	return token
}

func (this *spacemap) Delete(key string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, ok := this.tokens[key]
	if ok {
		delete(this.miners, v.publicKey)
		delete(this.tokens, key)
	}
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

func (this *spacemap) IsExit(pubkey string) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	_, ok := this.miners[pubkey]
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

func UpdateSpacemap(key, ip, fid string) string {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	v, _ := sm.miners[key]
	info, ok2 := sm.tokens[v]
	if ok2 {
		info.updateTime = time.Now().Unix()
		info.fillerId = fid
		sm.tokens[v] = info
		return v
	}
	token := tools.RandStr(16)
	data := authspaceinfo{}
	data.publicKey = key
	data.ip = ip
	data.fillerId = fid
	data.updateTime = time.Now().Unix()
	sm.miners[key] = token
	sm.tokens[token] = data
	return token
}
