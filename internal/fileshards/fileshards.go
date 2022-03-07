package fileshards

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"scheduler-mining/tools"
	"strconv"

	"github.com/klauspost/reedsolomon"
	"github.com/pkg/errors"
)

func cutFileRule(file string) (int64, int64, uint8, error) {
	f, err := os.Stat(file)
	if err != nil {
		return 0, 0, 0, err
	}
	if f.IsDir() {
		return 0, 0, 0, errors.Errorf("[%v] is not a file", file)
	}
	fmt.Println(f.Size())
	num := f.Size() / (1024 * 1024 * 1024)
	slicesize := f.Size() / (num + 1)
	tailsize := f.Size() - slicesize*(num+1)
	return slicesize, slicesize + tailsize, uint8(num) + 1, nil
}

func CutFile(file string) error {
	slicesize, lastslicesize, num, err := cutFileRule(file)
	if err != nil {
		return err
	}
	fi, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fi.Close()
	b := make([]byte, slicesize)
	lb := make([]byte, lastslicesize)
	var i int64 = 1
	for ; i <= int64(num); i++ {
		fi.Seek((i-1)*(slicesize), 0)
		f, err := os.OpenFile("./"+fi.Name()+"-"+strconv.Itoa(int(i)), os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return err
		}
		if i == int64(num) {
			fi.Read(lb)
			f.Write(lb)
			f.Close()
		} else {
			fi.Read(b)
			f.Write(b)
			f.Close()
		}
	}
	return nil
}

func Shards(inFilePath, outFilePath string, dataShards, rduShards int) (int, []string, error) {
	var shardSize = 0
	var shardsname = make([]string, 0)
	// Create encoding matrix.
	enc, err := reedsolomon.New(dataShards, rduShards)
	if err != nil {
		return 0, shardsname, errors.Wrap(err, "reedsolomon.New err")
	}

	b, err := ioutil.ReadFile(inFilePath)
	if err != nil {
		return 0, shardsname, errors.Wrap(err, "ioutil.ReadFile err")
	}

	// Split the file into equally sized shards.
	shards, err := enc.Split(b)
	if err != nil {
		return 0, shardsname, errors.Wrap(err, "enc.Split err")
	}
	shardSize = len(shards[0])
	// Encode parity
	err = enc.Encode(shards)
	if err != nil {
		return 0, shardsname, errors.Wrap(err, "enc.Encode err")
	}
	hashname := ""
	rdunum := 0
	// Write out the resulting files.
	_, file := filepath.Split(inFilePath)
	for i, shard := range shards {
		hashname = ""
		outfn := fmt.Sprintf("%s.%d", file, i)
		shardfilepath := filepath.Join(outFilePath, outfn)
		//fmt.Println("Writing to", outfn)
		err = ioutil.WriteFile(shardfilepath, shard, os.ModePerm)
		if err != nil {
			return 0, shardsname, errors.Wrap(err, "ioutil.WriteFile")
		}
		shardhash, err := tools.CalcFileHash(shardfilepath)
		if err != nil {
			return 0, shardsname, errors.Wrap(err, "CalcFileHash")
		}
		hashname = shardhash + fmt.Sprintf(".%d", i)
		if (i + 1) > dataShards {
			hashname = shardhash + fmt.Sprintf(".r%v", rdunum)
			rdunum++
		}
		var shardnewname = filepath.Join(outFilePath, hashname)
		err = os.Rename(shardfilepath, shardnewname)
		if err != nil {
			return 0, shardsname, errors.Wrap(err, "Rename")
		}
		shardsname = append(shardsname, shardnewname)
	}
	return shardSize, shardsname, nil
}
