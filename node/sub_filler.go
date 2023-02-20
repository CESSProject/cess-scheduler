/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package node

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/proof"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/pkg/errors"
)

type PubRSARequest struct {
	FilePath    string `json:"file_path"`
	BlockSize   int64  `json:"block_size"`
	CallbackUrl string `json:"callback_url"`
}

type StorageTagType struct {
	T           proof.T
	Phi         []proof.Sigma `json:"phi"`
	SigRootHash []byte        `json:"sig_root_hash"`
	E           string        `json:"e"`
	N           string        `json:"n"`
}

// task_GenerateFiller is used to generate filler
// and store it in the channel for standby
func (n *Node) task_GenerateFiller(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	var (
		err        error
		fillerpath string
		fillerEle  Filler
	)
	n.Logs.GenFiller("info", errors.New(">>> Start task_GenerateFiller <<<"))
	for {
		for len(C_Filler) < configs.Num_Filler_Reserved {
			n.Logs.GenFiller("info", fmt.Errorf("A filler will be generated..."))
			// Generate filler
			fillerpath, err = n.GenerateFiller()
			if err != nil {
				n.Logs.GenFiller("err", err)
				continue
			}

			n.Logs.GenFiller("info", fmt.Errorf("A filler is generated: %v", filepath.Base(fillerpath)))
			// Calculate filler tag
			err = n.RequestAndSaveTag(fillerpath, fillerpath+configs.TagFileExt)
			if err != nil {
				n.Logs.GenFiller("err", err)
				os.Remove(fillerpath)
				continue
			}

			n.Logs.GenFiller("info", fmt.Errorf("Calculate the tag of the filler: %v", filepath.Base(fillerpath)+configs.TagFileExt))
			// save filler metainfo to channel
			fillerEle.Hash = filepath.Base(fillerpath)
			fillerEle.FillerPath = fillerpath
			fillerEle.TagPath = fillerpath + configs.TagFileExt
			C_Filler <- fillerEle
			n.Logs.GenFiller("info", fmt.Errorf("Produced a filler: %v", filepath.Base(fillerpath)))
		}
		time.Sleep(time.Second)
	}
}

func (n *Node) GenerateFiller() (string, error) {
	var (
		err        error
		fpath      string
		newPath    string
		fillerHash string
	)

	for {
		fpath = filepath.Join(n.FillerDir, fmt.Sprintf("%v", time.Now().UnixNano()))
		_, err = os.Stat(fpath)
		if err == nil {
			time.Sleep(time.Second)
			continue
		}
		break
	}

	defer os.Remove(fpath)

	// generate filler
	err = generateFiller(fpath, configs.FillerSize)
	if err != nil {
		return fpath, err
	}

	// judge filler size
	fstat, _ := os.Stat(fpath)
	if fstat.Size() != configs.FillerSize {
		return fpath, errors.New("size error")
	}

	// calc filler hash
	fillerHash, err = utils.CalcPathSHA256(fpath)
	if err != nil {
		return fpath, err
	}

	// rename filler
	newPath = filepath.Join(n.FillerDir, fillerHash)
	err = os.Rename(fpath, newPath)
	if err != nil {
		return fpath, err
	}
	return newPath, nil
}

func (n *Node) RequestAndSaveTag(fpath, tagpath string) error {
	var (
		err        error
		callTagUrl string
		tag        PoDR2PubData
		tagSave    StorageTagType
	)
	callTagUrl = fmt.Sprintf("%s:%d%s", configs.Localhost, n.Confile.GetSgxPort(), configs.GetTagRoute)
	err = getTagReq(fpath, configs.BlockSize, int64(configs.SgxCallBackPort), callTagUrl, configs.GetTagRoute_Callback, n.Confile.GetServiceAddr())
	if err != nil {
		return err
	}
	timeout := time.NewTicker(configs.TimeOut_WaitTag)
	defer timeout.Stop()

	select {
	case <-timeout.C:
		return fmt.Errorf("Timeout waiting for calculation tag")
	case tag = <-Ch_Tag:
	}

	// matrix, num, err := proof.SplitV2(fillerpath, configs.BlockSize)
	// if err != nil {
	// 	n.Logs.GenFiller("err", err)
	// 	continue
	// }

	u, err := base64.StdEncoding.DecodeString(tag.Result.T.Tag.U)
	if err != nil {
		return err
	}

	name, err := base64.StdEncoding.DecodeString(tag.Result.T.Tag.Name)
	if err != nil {
		return err
	}
	tagSave.T.Tag.U = u
	tagSave.T.Tag.Name = name
	tagSave.T.Tag.N = tag.Result.T.Tag.N

	//tag signature
	sig, err := hex.DecodeString(tag.Result.T.SigAbove)
	if err != nil {
		return err
	}
	tagSave.T.SigAbove = sig

	//phi
	for _, v := range tag.Result.Phi {
		phi_big, _ := new(big.Int).SetString(v, 10)
		tagSave.Phi = append(tagSave.Phi, phi_big.Bytes())
	}

	//sig_root_hash
	sig_root_hash, err := hex.DecodeString(tag.Result.SigRootHash)
	if err != nil {
		return err
	}
	tagSave.SigRootHash = sig_root_hash

	tagSave.E = tag.Result.Spk.E
	tagSave.N = tag.Result.Spk.N

	// filler tag
	err = saveTagToFile(tagpath, tagSave)
	if err != nil {
		os.Remove(tagpath)
		return err
	}

	//set public key
	_, err = proof.GetKey(n.Cache)
	if err != nil {
		n.Cache.Put([]byte(configs.SigKey_E), []byte(tag.Result.Spk.E))
		n.Cache.Put([]byte(configs.SigKey_N), []byte(tag.Result.Spk.N))
		proof.SetKey(n.Cache)
	}
	return nil
}

func getTagReq(fpath string, blocksize, callbackPort int64, callUrl, callbackRouter, callbackIp string) error {
	callbackurl := fmt.Sprintf("http://%v:%d%v", callbackIp, callbackPort, callbackRouter)
	param := PubRSARequest{
		FilePath:    configs.SgxMappingPath + fpath,
		BlockSize:   blocksize,
		CallbackUrl: callbackurl,
	}
	data, err := json.Marshal(param)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, callUrl, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	cli := http.Client{
		Transport: configs.GlobalTransport,
	}

	_, err = cli.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func generateFiller(fpath string, fsize uint64) error {
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	rows := fsize / configs.FillerLineLength
	for i := uint64(0); i < rows; i++ {
		f.WriteString(utils.RandStr(configs.FillerLineLength-1) + "\n")
	}
	err = f.Sync()
	if err != nil {
		os.Remove(fpath)
		return err
	}
	return nil
}

func saveTagToFile(fpath string, tag StorageTagType) error {
	tagFs, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer tagFs.Close()

	tagData, err := json.Marshal(&tag)
	if err != nil {
		return err
	}
	_, err = tagFs.Write(tagData)
	if err != nil {
		return err
	}

	return tagFs.Sync()
}
