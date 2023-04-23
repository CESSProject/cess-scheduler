package node

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/pkg/errors"
)

// task_ Common is used to judge whether the balance of
// your wallet meets the operation requirements.
func (node *Node) task_TraceFile(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			node.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	var record RecordFileInfo
	node.Logs.Upfile("info", errors.New(">>> Start task_TraceFile <<<"))

	for {
		time.Sleep(time.Minute)
		tracks, err := utils.DirFiles(node.TraceDir, 0)
		if err != nil {
			node.Logs.Upfile("err", fmt.Errorf("DirFiles: %v", err))
			continue
		}
		for i := 0; i < len(tracks); i++ {
			b, err := os.ReadFile(tracks[i])
			if err != nil {
				node.Logs.Upfile("err", fmt.Errorf("ReadFile: %v", err))
				continue
			}
			err = json.Unmarshal(b, &record)
			if err != nil {
				node.Logs.Upfile("err", fmt.Errorf("Unmarshal err: %v", err))
				continue
			}

			//Judge whether the file has been uploaded
			fileState, err := GetFileState(node.Chain, record.FileId)
			if err != nil {
				node.Logs.Upfile("err", fmt.Errorf("GetFileState err: %v", err))
				continue
			}

			// file state
			if fileState == chain.FILE_STATE_ACTIVE {
				os.Remove(tracks[i])
				continue
			}

			err = node.TrackFileBackupManagement(record.FileId, int64(record.FileSize), record.Chunks)
			node.Logs.Upfile("info", fmt.Errorf("TrackFileBackupManagement: %v", err))
		}

		files, err := utils.DirFiles(node.FileDir, 0)
		if err != nil {
			node.Logs.Upfile("err", fmt.Errorf("DirFiles: %v", err))
			continue
		}
		var filesmap = make(map[string][]string)
		for i := 0; i < len(files); i++ {
			fid := filepath.Base(files[i])
			filesmap[fid] = append(filesmap[fid], files[i])
		}

		for k, v := range filesmap {
			//Judge whether the file has been uploaded
			fileState, err := GetFileState(node.Chain, k)
			if err != nil {
				node.Logs.Upfile("err", fmt.Errorf("GetFileState err: %v", err))
				continue
			}

			// file state
			if fileState == chain.FILE_STATE_ACTIVE {
				continue
			}

			fstat, err := os.Stat(v[0])
			if err != nil {
				node.Logs.Upfile("info", fmt.Errorf("os.Stat err: %v", err))
				continue
			}
			err = node.TrackFileBackupManagement(k, int64(fstat.Size()*int64(len(v))), v)
			node.Logs.Upfile("info", fmt.Errorf("TrackFileBackupManagement: %v", err))
		}
	}
}
