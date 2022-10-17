package pattern

// import (
// 	"strings"
// 	"sync"
// )

// type minertoken struct {
// 	lock        *sync.Mutex
// 	minerTokens map[string]string
// }

// type authspaceinfo struct {
// 	publicKey  string
// 	ip         string
// 	fillerId   string
// 	updateTime int64
// }

// var mtoken *minertoken

// func init() {
// 	mtoken = new(minertoken)
// 	mtoken.lock = new(sync.Mutex)
// 	mtoken.minerTokens = make(map[string]string, 10)
// 	//spacem.tokens = make(map[string]authspaceinfo, 10)
// }

// func PutMinerToken(pkey, token, fid string) {
// 	mtoken.lock.Lock()
// 	mtoken.minerTokens[pkey] = token + "#" + fid
// 	mtoken.lock.Unlock()
// }

// func GetMinerToken(pkey string) (string, string) {
// 	mtoken.lock.Lock()
// 	tk := mtoken.minerTokens[pkey]
// 	mtoken.lock.Unlock()
// 	if tk != "" {
// 		data := strings.Split(tk, "#")
// 		if len(data) == 2 {
// 			return data[0], data[1]
// 		}
// 	}
// 	return "", ""
// }

// // func VerifySpaceToken(token string) (string, string, string, error) {
// // 	spacem.lock.Lock()
// // 	v, ok := spacem.tokens[token]
// // 	spacem.lock.Unlock()
// // 	if !ok {
// // 		return "", "", "", errors.New("Invalid token")
// // 	}
// // 	return v.publicKey, v.fillerId, v.ip, nil
// // }

// // func IsMaxSpacem(key string) bool {
// // 	spacem.lock.Lock()
// // 	defer spacem.lock.Unlock()
// // 	_, ok := spacem.miners[key]
// // 	if !ok {
// // 		if len(spacem.miners) <= 30 {
// // 			return false
// // 		}
// // 		return true
// // 	}
// // 	return false
// // }

// // func UpdateSpacemap(key, ip, fid string) string {
// // 	spacem.lock.Lock()
// // 	defer spacem.lock.Unlock()

// // 	v, _ := spacem.miners[key]
// // 	info, ok2 := spacem.tokens[v]
// // 	if ok2 {
// // 		info.updateTime = time.Now().Unix()
// // 		info.fillerId = fid
// // 		spacem.tokens[v] = info
// // 		return v
// // 	}
// // 	token := utils.RandStr(16)
// // 	data := authspaceinfo{}
// // 	data.publicKey = key
// // 	data.ip = ip
// // 	data.fillerId = fid
// // 	data.updateTime = time.Now().Unix()
// // 	spacem.miners[key] = token
// // 	spacem.tokens[token] = data
// // 	return token
// // }

// // func DeleteSpacemap(key string) {
// // 	spacem.lock.Lock()
// // 	v, ok := spacem.tokens[key]
// // 	if ok {
// // 		delete(spacem.miners, v.publicKey)
// // 		delete(spacem.tokens, key)
// // 	}
// // 	spacem.lock.Unlock()
// // }

// // func DeleteExpiredSpacem() {
// // 	spacem.lock.Lock()
// // 	for k, v := range spacem.tokens {
// // 		if time.Since(time.Unix(v.updateTime, 0)).Minutes() > 10 {
// // 			delete(spacem.miners, v.publicKey)
// // 			delete(spacem.tokens, k)
// // 		}
// // 	}
// // 	spacem.lock.Unlock()
// // }

// // func GetConnsMinerNum() int {
// // 	spacem.lock.Lock()
// // 	defer spacem.lock.Unlock()
// // 	return len(spacem.tokens)
// // }

// // func IsExitSpacem(pubkey string) bool {
// // 	spacem.lock.Lock()
// // 	_, ok := spacem.miners[pubkey]
// // 	spacem.lock.Unlock()
// // 	return ok
// // }

// // func GetConnectedSpacem() []string {
// // 	var data = make([]string, 0)
// // 	spacem.lock.Lock()
// // 	for k, _ := range spacem.miners {
// // 		addr, _ := utils.EncodePublicKeyAsCessAccount([]byte(k))
// // 		data = append(data, addr)
// // 	}
// // 	spacem.lock.Unlock()
// // 	return data
// // }
