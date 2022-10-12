package pattern

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type BaseFillerList struct {
	Lock       *sync.Mutex
	BaseFiller map[string]FillerData
}

type FillerData struct {
	BaseMinerIp   string
	FillerId      string
	UpdateTime    int64
	CopyMinerPkey []string
}

type BaseFiller struct {
	MinerIp  []string `json:"minerIp"`
	FillerId string   `json:"fillerId"`
}

var basefiller *BaseFillerList

func init() {
	basefiller = new(BaseFillerList)
	basefiller.Lock = new(sync.Mutex)
	basefiller.BaseFiller = make(map[string]FillerData, 0)
}

func AddBaseFiller(baseMinerPkey, baseMinerIp, fillerid string) {
	basefiller.Lock.Lock()
	var fillerData FillerData
	fillerData.BaseMinerIp = baseMinerIp
	fillerData.FillerId = fillerid
	fillerData.UpdateTime = time.Now().Unix()
	fillerData.CopyMinerPkey = make([]string, 0)
	basefiller.BaseFiller[baseMinerPkey] = fillerData
	basefiller.Lock.Unlock()
}

func AppendBaseFiller(basePkey, pkey string) {
	basefiller.Lock.Lock()
	v, ok := basefiller.BaseFiller[basePkey]
	if ok {
		v.CopyMinerPkey = append(v.CopyMinerPkey, pkey)
		basefiller.BaseFiller[basePkey] = v
	}
	basefiller.Lock.Unlock()
}

func HasBaseFiller(basePkey string) bool {
	basefiller.Lock.Lock()
	defer basefiller.Lock.Unlock()
	v, ok := basefiller.BaseFiller[basePkey]
	if ok {
		if len(v.CopyMinerPkey) == 0 || len(v.CopyMinerPkey) >= 3 {
			delete(basefiller.BaseFiller, basePkey)
			return false
		}
		return true
	}
	return false
}

func GetBaseFiller(pkey string) (string, string, error) {
	basefiller.Lock.Lock()
	defer basefiller.Lock.Unlock()

	var exist bool

	for k, v := range basefiller.BaseFiller {
		exist = false
		if len(v.CopyMinerPkey) < 3 {
			for _, v2 := range v.CopyMinerPkey {
				if v2 == pkey {
					exist = true
					break
				}
			}
		} else {
			delete(basefiller.BaseFiller, k)
			continue
		}
		if !exist {
			v.CopyMinerPkey = append(v.CopyMinerPkey, pkey)
			if len(v.CopyMinerPkey) >= 3 {
				delete(basefiller.BaseFiller, k)
			} else {
				basefiller.BaseFiller[k] = v
			}
			fmt.Println(len(v.CopyMinerPkey), " -- ", []byte(k))
			return v.FillerId, v.BaseMinerIp, nil
		}
	}
	return "", "", errors.New("empty")
}

func GetBaseFillerLength() int {
	basefiller.Lock.Lock()
	defer basefiller.Lock.Unlock()
	return len(basefiller.BaseFiller)
}

func DelereBaseFiller(minerPkey string) {
	basefiller.Lock.Lock()
	for k, _ := range basefiller.BaseFiller {
		if k == minerPkey {
			delete(basefiller.BaseFiller, k)
			break
		}
	}
	basefiller.Lock.Unlock()
}
