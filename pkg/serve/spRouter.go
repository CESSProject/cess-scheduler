package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	Backups     []map[int]string `json:"backups,omitempty"`
}

type StorageProgressRouter struct {
	BaseRouter
	Cach db.Cacher
}

type MsgStorageProgress struct {
	RootHash string `json:"roothash"`
}

// AuthRouter Handle
func (s *StorageProgressRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call StorageProgressRouter Handle")
	fmt.Println("recv from client : msgId=", request.GetMsgID())
	if request.GetMsgID() != Msg_Progress {
		fmt.Println("MsgId error")
		ctx()
		return
	}

	remote := request.GetConnection().RemoteAddr().String()
	val, err := s.Cach.Get([]byte(remote))
	if err != nil {
		s.Cach.Put([]byte(remote), utils.Int64ToBytes(time.Now().Unix()))
	} else {
		if time.Since(time.Unix(utils.BytesToInt64(val), 0)).Seconds() < 3 {
			ctx()
			return
		} else {
			s.Cach.Delete([]byte(remote))
		}
	}

	var msg MsgStorageProgress
	err = json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		ctx()
		return
	}

	val, err = s.Cach.Get([]byte(msg.RootHash))
	if err != nil {
		request.GetConnection().SendMsg(Msg_ServerErr, nil)
		return
	}

	err = request.GetConnection().SendMsg(Msg_OK, val)
	if err != nil {
		ctx()
		return
	}
}
