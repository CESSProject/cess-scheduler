package fileshards

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/klauspost/reedsolomon"
	"github.com/pkg/errors"
)

func Shards(inFilePath, outFilePath string, dataShards, rduShards int) (int, error) {
	var shardSize = 0
	// Create encoding matrix.
	enc, err := reedsolomon.New(dataShards, rduShards)
	if err != nil {
		return 0, errors.Wrap(err, "reedsolomon.New err")
	}

	b, err := ioutil.ReadFile(inFilePath)
	if err != nil {
		return 0, errors.Wrap(err, "ioutil.ReadFile err")
	}

	// Split the file into equally sized shards.
	shards, err := enc.Split(b)
	if err != nil {
		return 0, errors.Wrap(err, "enc.Split err")
	}
	shardSize = len(shards[0])
	// Encode parity
	err = enc.Encode(shards)
	if err != nil {
		return 0, errors.Wrap(err, "enc.Encode err")
	}

	// Write out the resulting files.
	_, file := filepath.Split(inFilePath)
	for i, shard := range shards {
		outfn := fmt.Sprintf("%s.%d", file, i)

		//fmt.Println("Writing to", outfn)
		err = ioutil.WriteFile(filepath.Join(outFilePath, outfn), shard, os.ModePerm)
		if err != nil {
			return 0, errors.Wrap(err, "ioutil.WriteFile")
		}
	}
	return shardSize, nil
}
