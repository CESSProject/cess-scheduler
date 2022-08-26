package pattern

import (
	"cess-scheduler/internal/chain"
	"sync"
)

type Fillermetamap struct {
	lock        *sync.Mutex
	Fillermetas map[string][]chain.FillerMetaInfo
}

var FillerMap *Fillermetamap

func init() {
	FillerMap = new(Fillermetamap)
	FillerMap.Fillermetas = make(map[string][]chain.FillerMetaInfo)
	FillerMap.lock = new(sync.Mutex)
}

func (this *Fillermetamap) Add(pubkey string, data chain.FillerMetaInfo) {
	this.lock.Lock()
	defer this.lock.Unlock()
	_, ok := this.Fillermetas[pubkey]
	if !ok {
		this.Fillermetas[pubkey] = make([]chain.FillerMetaInfo, 0)
	}
	this.Fillermetas[pubkey] = append(this.Fillermetas[pubkey], data)
}

func (this *Fillermetamap) GetNum(pubkey string) int {
	this.lock.Lock()
	defer this.lock.Unlock()

	return len(this.Fillermetas[pubkey])
}

func (this *Fillermetamap) Delete(pubkey string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	delete(this.Fillermetas, pubkey)
}

func (this *Fillermetamap) Lock() {
	this.lock.Lock()
}

func (this *Fillermetamap) UnLock() {
	this.lock.Unlock()
}
