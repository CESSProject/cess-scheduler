package fileshards

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"scheduler-mining/tools"

	"github.com/klauspost/reedsolomon"
	"github.com/pkg/errors"
)

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
