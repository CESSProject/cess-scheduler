package pattern

import (
	"sync"
	"time"

	"github.com/CESSProject/cess-scheduler/tools"
)

type Protect struct {
	L         *sync.Mutex
	Reqmap    map[string]int64
	Blacklist map[string]int64
}

var p *Protect

func init() {
	p = new(Protect)
	p.L = new(sync.Mutex)
	p.Reqmap = make(map[string]int64, 10)
	p.Blacklist = make(map[string]int64, 10)
}

func DeleteBliacklist(key string) {
	p.L.Lock()
	delete(p.Blacklist, key)
	p.L.Unlock()
}

func AddBlacklist(key string) {
	p.L.Lock()
	p.Blacklist[key] = time.Now().Unix()
	p.L.Unlock()
}

func IsPass(key string) bool {
	p.L.Lock()
	defer p.L.Unlock()

	v, ok := p.Blacklist[key]
	if ok {
		if time.Since(time.Unix(v, 0)).Minutes() > 10 {
			delete(p.Blacklist, key)
			return true
		}
		return false
	}
	v, ok = p.Reqmap[key]
	if !ok {
		p.Reqmap[key] = time.Now().Unix()
	} else {
		if time.Since(time.Unix(v, 0)).Seconds() <= 3 {
			p.Blacklist[key] = time.Now().Unix()
			delete(p.Reqmap, key)
			return false
		} else {
			p.Reqmap[key] = time.Now().Unix()
		}
	}
	return true
}

func GetBlacklist() []string {
	var data = make([]string, 0)
	p.L.Lock()
	for k, _ := range p.Blacklist {
		addr, _ := tools.EncodeToCESSAddr([]byte(k))
		data = append(data, addr)
	}
	p.L.Unlock()
	return data
}
