package fileHandling

import (
	"cess-scheduler/configs"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"github.com/klauspost/reedsolomon"
)

func reedSolomonRule(size int64) (int, int, error) {
	fsize := size
	datachunk := int64(1)
	if fsize <= 1 {
		return 1, 0, nil
	}

	if fsize <= 8*configs.SIZE_1GB {
		if fsize <= 4*configs.SIZE_1GB {
			if fsize <= configs.SIZE_1MB {
				datachunk = 2
				goto result
			}

			if fsize <= configs.SIZE_1GB {
				datachunk = 4
				goto result
			}

			if fsize <= 2*configs.SIZE_1GB {
				datachunk = 6
				goto result
			}

			datachunk = 8
			goto result
		}
		if fsize <= 6*configs.SIZE_1GB {
			datachunk = 10
			goto result
		}
		datachunk = 12
		goto result
	}

	if fsize <= 10*configs.SIZE_1GB {
		datachunk = 14
		goto result
	}

	if fsize <= 12*configs.SIZE_1GB {
		datachunk = 16
		goto result
	}

	if fsize <= 14*configs.SIZE_1GB {
		datachunk = 18
		goto result
	}

	if fsize <= 16*configs.SIZE_1GB {
		datachunk = 20
		goto result
	}

	datachunk = 30
result:

	rdchunks := datachunk / 2

	if math.Ceil(float64(fsize)/float64(datachunk)*float64(datachunk+rdchunks)) > float64(fsize)*float64(1.5) {
		datachunk -= 1
	}
	return int(datachunk), int(datachunk / 2), nil
}

func ReedSolomon(fpath string, size int64) ([]string, int, int, error) {
	var shardspath = make([]string, 0)
	datashards, rdunshards, err := reedSolomonRule(size)
	if err != nil {
		return shardspath, datashards, rdunshards, err
	}

	if datashards+rdunshards <= 9 {
		enc, err := reedsolomon.New(datashards, rdunshards)
		if err != nil {
			return shardspath, datashards, rdunshards, err
		}

		b, err := ioutil.ReadFile(fpath)
		if err != nil {
			return shardspath, datashards, rdunshards, err
		}

		// Split the file into equally sized shards.
		shards, err := enc.Split(b)
		if err != nil {
			return shardspath, datashards, rdunshards, err
		}
		// Encode parity
		err = enc.Encode(shards)
		if err != nil {
			return shardspath, datashards, rdunshards, err
		}
		// Write out the resulting files.
		for i, shard := range shards {
			var outfn = fmt.Sprintf("%s.00%d", fpath, i)
			err = ioutil.WriteFile(outfn, shard, os.ModePerm)
			if err != nil {
				return shardspath, datashards, rdunshards, err
			}
			shardspath = append(shardspath, outfn)
		}
		return shardspath, datashards, rdunshards, nil
	}

	// Create encoding matrix.
	enc, err := reedsolomon.NewStream(datashards, rdunshards)
	if err != nil {
		return shardspath, datashards, rdunshards, err
	}

	f, err := os.Open(fpath)
	if err != nil {
		return shardspath, datashards, rdunshards, err
	}

	instat, err := f.Stat()
	if err != nil {
		return shardspath, datashards, rdunshards, err
	}

	shards := datashards + rdunshards
	out := make([]*os.File, shards)

	// Create the resulting files.
	dir, file := filepath.Split(fpath)

	for i := range out {
		var outfn string
		if i < 10 {
			outfn = fmt.Sprintf("%s.00%d", file, i)
		} else {
			outfn = fmt.Sprintf("%s.0%d", file, i)
		}
		out[i], err = os.Create(filepath.Join(dir, outfn))
		if err != nil {
			return shardspath, datashards, rdunshards, err
		}
		out[i].Close()
		shardspath = append(shardspath, filepath.Join(dir, outfn))
	}

	// Split into files.
	data := make([]io.Writer, datashards)
	for i := range data {
		data[i] = out[i]
	}
	// Do the split
	err = enc.Split(f, data, instat.Size())
	if err != nil {
		return shardspath, datashards, rdunshards, err
	}

	// Close and re-open the files.
	input := make([]io.Reader, datashards)

	for i := range data {
		f, err := os.Open(out[i].Name())
		if err != nil {
			return shardspath, datashards, rdunshards, err
		}
		input[i] = f
		defer f.Close()
	}

	// Create parity output writers
	parity := make([]io.Writer, rdunshards)
	for i := range parity {
		parity[i] = out[datashards+i]
		defer out[datashards+i].Close()
	}

	// Encode parity
	err = enc.Encode(input, parity)
	if err != nil {
		return shardspath, datashards, rdunshards, err
	}

	return shardspath, datashards, rdunshards, nil
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
