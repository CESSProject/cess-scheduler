package fileauth

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/HaoyuHu/gosimhash"
	"github.com/HaoyuHu/gosimhash/utils"
	"github.com/pkg/errors"
)

const (
	dictPath     = "./dict/jieba.dict.utf8"
	hmmPath      = "./dict/hmm_model.utf8"
	userDictPath = "./dict/user.dict.utf8"
	idfPath      = "./dict/idf.utf8"
	stopwordPath = "./dict/stop_words.utf8"
)

func getTextSimhash(filename string) (string, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return "", errors.Wrap(err, "stat err")
	}
	if fi.IsDir() {
		return "", errors.New("Not a file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return "", errors.Wrap(err, "open err")
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return "", errors.Wrap(err, "ReadAll err")
	}
	hasher := gosimhash.NewSimhasher(utils.NewJenkinsHasher(), dictPath, hmmPath, userDictPath, idfPath, stopwordPath)
	defer hasher.Free()

	sentence := string(content)
	//fmt.Println(fi.Size())
	var topN int = int(fi.Size()) / 3
	if topN > 1000 {
		topN = 1000
	}
	simhash := hasher.MakeSimhash(&sentence, topN)
	if simhash == 0 {
		return "", errors.New("Unsupported file format")
	}
	return fmt.Sprintf("%v", simhash), nil
}
