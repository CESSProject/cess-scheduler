package fileshards

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"scheduler-mining/configs"
	"scheduler-mining/tools"
	"strconv"

	"github.com/klauspost/reedsolomon"
	"github.com/pkg/errors"
)

func cutFileRule(file string) (uint64, uint64, uint8, error) {
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
	return uint64(slicesize), uint64(slicesize + tailsize), uint8(num) + 1, nil
}

func CutDataRule(size uint64) (uint64, uint64, uint8, error) {
	fmt.Println(size)
	num := size / (1024 * 1024 * 1024)
	slicesize := size / (num + 1)
	tailsize := size - slicesize*(num+1)
	return uint64(slicesize), uint64(slicesize + tailsize), uint8(num) + 1, nil
}

func CutFile(file string) ([]string, uint64, uint64, error) {
	var fileshards = make([]string, 0)
	slicesize, lastslicesize, num, err := cutFileRule(file)
	if err != nil {
		return nil, 0, 0, err
	}
	fi, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, 0, 0, err
	}
	defer fi.Close()
	//dir := filepath.Dir(file)
	b := make([]byte, slicesize)
	lb := make([]byte, lastslicesize)
	var i uint64 = 0
	for ; i < uint64(num); i++ {
		fi.Seek(int64((i)*(slicesize)), 0)
		var shards = fi.Name() + "-" + strconv.Itoa(int(i))
		fmt.Println("shards: ", shards)
		f, err := os.OpenFile(shards, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return nil, 0, 0, err
		}
		fileshards = append(fileshards, shards)
		if i == uint64(num) {
			fi.Read(lb)
			f.Write(lb)
			f.Close()
		} else {
			fi.Read(b)
			f.Write(b)
			f.Close()
		}
	}
	return fileshards, slicesize, lastslicesize, nil
}

func reedSolomonRule(file string) (int, int, error) {
	f, err := os.Stat(file)
	if err != nil {
		return 0, 0, err
	}
	num := int(f.Size() / int64(configs.MinSegMentSize))
	if num < 1 {
		return 1, 1, nil
	} else if num/4 <= 0 {
		return num + 1, 1, nil
	} else {
		return num + 1, num / 4, nil
	}
}

func ReedSolomon(file string) ([]string, int, int, error) {
	var shardsname = make([]string, 0)
	datashards, rdunshards, err := reedSolomonRule(file)
	if err != nil {
		return shardsname, datashards, rdunshards, err
	}

	enc, err := reedsolomon.New(datashards, rdunshards)
	if err != nil {
		return shardsname, datashards, rdunshards, err
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		return shardsname, datashards, rdunshards, err
	}

	// Split the file into equally sized shards.
	shards, err := enc.Split(b)
	if err != nil {
		return shardsname, datashards, rdunshards, err
	}
	// Encode parity
	err = enc.Encode(shards)
	if err != nil {
		return shardsname, datashards, rdunshards, err
	}
	// Write out the resulting files.
	for i, shard := range shards {
		var outfn = fmt.Sprintf("%s-%d", file, i)
		//shardfilepath := filepath.Join(outFilePath, outfn)
		fmt.Println("Writing to", outfn)
		err = ioutil.WriteFile(outfn, shard, os.ModePerm)
		if err != nil {
			return shardsname, datashards, rdunshards, err
		}
		shardsname = append(shardsname, outfn)
	}
	return shardsname, datashards, rdunshards, nil
}

func ReedSolomon_Restore(file string, datashards, rdushards int) error {
	enc, err := reedsolomon.New(datashards, rdushards)
	if err != nil {
		return err
	}
	shards := make([][]byte, datashards+rdushards)
	for i := range shards {
		infn := fmt.Sprintf("%s-%d", file, i)
		fmt.Println("Opening", infn)
		shards[i], err = ioutil.ReadFile(infn)
		if err != nil {
			fmt.Println("Error reading file", err)
			shards[i] = nil
		}
	}

	// Verify the shards
	ok, _ := enc.Verify(shards)
	if ok {
		fmt.Println("No reconstruction needed")
	} else {
		fmt.Println("Verification failed. Reconstructing data")
		err = enc.Reconstruct(shards)
		if err != nil {
			fmt.Println("Reconstruct failed -", err)
			return err
		}
		ok, err = enc.Verify(shards)
		if !ok {
			fmt.Println("Verification failed after reconstruction, data likely corrupted.")
			return err
		}
	}
	fmt.Println("Writing data to", file)
	f, err := os.Create(file)
	if err != nil {
		return err
	}

	// We don't know the exact filesize.
	err = enc.Join(f, shards, len(shards[0])*datashards)
	if err != nil {
		return err
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
