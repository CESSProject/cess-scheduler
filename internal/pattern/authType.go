package pattern

import (
	"errors"
	"sync"
	"time"
)

type Authmap struct {
	lock   *sync.Mutex
	miners map[string]string
	tokens map[string]Authinfo
}

type Authinfo struct {
	PublicKey  string
	FileId     string
	FileName   string
	UpdateTime int64
	BlockTotal uint32
}

var auth *Authmap

func init() {
	auth = new(Authmap)
	auth.lock = new(sync.Mutex)
	auth.miners = make(map[string]string, 10)
	auth.tokens = make(map[string]Authinfo, 10)
}

func UpdateAuth(addr string) string {
	auth.lock.Lock()
	v1, ok := auth.miners[addr]
	if ok {
		v2, ok := auth.tokens[v1]
		if ok {
			v2.UpdateTime = time.Now().Unix()
			auth.tokens[v1] = v2
			return v1
		}
		delete(auth.miners, addr)
	}
	auth.lock.Unlock()
	return ""
}

func AddAuth(addr, token string, info Authinfo) {
	auth.lock.Lock()
	auth.miners[addr] = token
	auth.tokens[token] = info
	auth.lock.Unlock()
}

func IsToken(token string) bool {
	auth.lock.Lock()
	_, ok := auth.tokens[token]
	auth.lock.Unlock()
	return ok
}

func DeleteAuth(token string) {
	auth.lock.Lock()
	v, ok := auth.tokens[token]
	if ok {
		delete(auth.miners, v.PublicKey)
		delete(auth.tokens, token)
	}
	auth.lock.Unlock()
}

func DeleteExpiredAuth() {
	auth.lock.Lock()
	for k, v := range auth.tokens {
		if time.Since(time.Unix(v.UpdateTime, 0)).Minutes() > 5 {
			delete(auth.miners, v.PublicKey)
			delete(auth.tokens, k)
		}
	}
	auth.lock.Unlock()
}

func GetAndUpdateAuth(key string) (uint32, string, string, string, error) {
	auth.lock.Lock()
	v, ok := auth.tokens[key]
	if !ok {
		return 0, "", "", "", errors.New("Authentication failed")
	}
	v.UpdateTime = time.Now().Unix()
	auth.tokens[key] = v
	auth.lock.Unlock()
	return v.BlockTotal, v.FileId, v.PublicKey, v.FileName, nil
}
