package pattern

import (
	"errors"
	"sync"
)

type BaseFillerList struct {
	Lock       *sync.Mutex
	BaseFiller map[string][]string
}

type BaseFiller struct {
	MinerIp  []string `json:"minerIp"`
	FillerId string   `json:"fillerId"`
}

var basefiller *BaseFillerList

func init() {
	basefiller = new(BaseFillerList)
	basefiller.Lock = new(sync.Mutex)
	basefiller.BaseFiller = make(map[string][]string, 10)
}

func AddBaseFiller(ip, fillerid string) {
	basefiller.Lock.Lock()
	_, ok := basefiller.BaseFiller[fillerid]
	if !ok {
		basefiller.BaseFiller[fillerid] = make([]string, 0)
		basefiller.BaseFiller[fillerid] = append(basefiller.BaseFiller[fillerid], ip)
	} else {
		basefiller.BaseFiller[fillerid] = append(basefiller.BaseFiller[fillerid], ip)
	}
	basefiller.Lock.Unlock()
}

func GetAndInsertBaseFiller(ip string) (string, string, error) {
	basefiller.Lock.Lock()
	defer basefiller.Lock.Unlock()

	var exist bool
	var del_count int
	if len(basefiller.BaseFiller) > 100 {
		for k, v := range basefiller.BaseFiller {
			if len(v) > 1 {
				delete(basefiller.BaseFiller, k)
			}
		}
		if len(basefiller.BaseFiller) > 100 {
			for k, _ := range basefiller.BaseFiller {
				delete(basefiller.BaseFiller, k)
				del_count++
				if del_count > 50 {
					break
				}
			}
		}
	}
	for k, v := range basefiller.BaseFiller {
		exist = false
		if len(v) < 3 {
			if len(v) == 0 {
				delete(basefiller.BaseFiller, k)
				continue
			}
			for _, v2 := range v {
				if v2 == ip {
					exist = true
					break
				}
			}
		} else {
			delete(basefiller.BaseFiller, k)
			continue
		}
		if !exist {
			basefiller.BaseFiller[k] = append(basefiller.BaseFiller[k], ip)
			return k, v[0], nil
		}
	}
	return "", "", errors.New("empty")
}

func GetBaseFillerLength() int {
	basefiller.Lock.Lock()
	defer basefiller.Lock.Unlock()
	return len(basefiller.BaseFiller)
}
