package fileauth

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/HaoyuHu/gosimhash"
	"github.com/pkg/errors"
)

func GetTextSimhash(filename string) (uint64, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return 0, errors.Wrap(err, "stat err")
	}
	if fi.IsDir() {
		return 0, errors.New("Not a file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return 0, errors.Wrap(err, "open err")
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return 0, errors.Wrap(err, "ReadAll err")
	}
	hasher := gosimhash.NewSimpleSimhasher()
	defer hasher.Free()

	sentence := string(content)
	fmt.Println(fi.Size())
	var topN int = int(fi.Size()) / 3
	if topN > 1000 {
		topN = 1000
	}
	simhash := hasher.MakeSimhash(&sentence, topN)
	if simhash == 0 {
		return 0, errors.New("Unsupported file format")
	}
	return simhash, nil
}
