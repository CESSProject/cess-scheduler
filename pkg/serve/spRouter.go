package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
)

type StorageProgress struct {
	FileId      string           `json:"file_id"`
	FileState   string           `json:"file_state"`
	Scheduler   string           `json:"scheduler"`
	FileSize    int64            `json:"file_size"`
	IsUpload    bool             `json:"is_upload"`
	IsCheck     bool             `json:"is_check"`
	IsShard     bool             `json:"is_shard"`
	IsScheduler bool             `json:"is_scheduler"`
	Backups     []map[int]string `json:"backups, omitempty"`
}

type StorageProgressRouter struct {
	BaseRouter
	Cache db.Cacher
}

type MsgStorageProgress struct {
	RootHash string `json:"roothash"`
}

// AuthRouter Handle
func (this *StorageProgressRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call AuthRouter Handle")
	fmt.Println("recv from client : msgId=", request.GetMsgID())
	if request.GetMsgID() != Msg_Auth {
		fmt.Println("MsgId error")
		ctx()
		return
	}

	remote := request.GetConnection().RemoteAddr().String()
	val, err := this.Cache.Get([]byte(remote))
	if err != nil {
		this.Cache.Put([]byte(remote), utils.Int64ToBytes(time.Now().Unix()))
	} else {
		if time.Since(time.Unix(utils.BytesToInt64(val), 0)).Minutes() < 1 {
			ctx()
			return
		} else {
			this.Cache.Delete([]byte(remote))
		}
	}

	var msg MsgAuth
	err = json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		ctx()
		return
	}

	puk, err := utils.DecodePublicKeyOfCessAccount(msg.Account)
	if err != nil {
		puk, err = utils.DecodePublicKeyOfSubstrateAccount(msg.Account)
		if err != nil {
			ctx()
			return
		}
	}

	ok, err := VerifySign(puk, []byte(msg.Msg), msg.Sign)
	if err != nil || !ok {
		ctx()
		return
	}

	token := utils.GetRandomcode(configs.TokenLength)
	err = request.GetConnection().SendMsg(Msg_OK, []byte(token))
	if err != nil {
		ctx()
		return
	}
	Tokens.Add(token)
}
