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

package erasure

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/klauspost/reedsolomon"
)

// ReedSolomon uses reed-solomon algorithm to redundancy files
// Return:
//
//  1. All file blocks (sorted sequentially)
//  2. Number of data blocks
//  3. Number of redundant blocks
//  4. Error message
func ReedSolomon(fpath string) ([]string, int, int, error) {
	var shardspath = make([]string, 0)
	fstat, err := os.Stat(fpath)
	if err != nil {
		return nil, 0, 0, err
	}
	datashards, rdunshards, err := reedSolomonRule(fstat.Size())
	if err != nil {
		return shardspath, datashards, rdunshards, err
	}

	if datashards+rdunshards <= 3 {
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
		out[i].Close()
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

// ReedSolomon_Restore uses reed-solomon algorithm to restore files
// which are located in the dir directory and named fid.
func ReedSolomon_Restore(dir, fid string, datashards, rdushards int) error {
	outfn := filepath.Join(dir, fid)
	if rdushards == 0 {
		return os.Rename(outfn+".000", outfn)
	}
	if datashards+rdushards <= 6 {
		enc, err := reedsolomon.New(datashards, rdushards)
		if err != nil {
			return err
		}
		shards := make([][]byte, datashards+rdushards)
		for i := range shards {
			infn := fmt.Sprintf("%s.00%d", outfn, i)
			shards[i], err = ioutil.ReadFile(infn)
			if err != nil {
				shards[i] = nil
			}
		}

		// Verify the shards
		ok, _ := enc.Verify(shards)
		if !ok {
			err = enc.Reconstruct(shards)
			if err != nil {
				return err
			}
			ok, err = enc.Verify(shards)
			if !ok {
				return err
			}
		}
		f, err := os.Create(outfn)
		if err != nil {
			return err
		}

		err = enc.Join(f, shards, len(shards[0])*datashards)
		return err
	}

	enc, err := reedsolomon.NewStream(datashards, rdushards)
	if err != nil {
		return err
	}

	// Open the inputs
	shards, size, err := openInput(datashards, rdushards, outfn)
	if err != nil {
		return err
	}

	// Verify the shards
	ok, err := enc.Verify(shards)
	if !ok {
		shards, size, err = openInput(datashards, rdushards, outfn)
		if err != nil {
			return err
		}

		out := make([]io.Writer, len(shards))
		for i := range out {
			if shards[i] == nil {
				var outfn string
				if i < 10 {
					outfn = fmt.Sprintf("%s.00%d", outfn, i)
				} else {
					outfn = fmt.Sprintf("%s.0%d", outfn, i)
				}
				out[i], err = os.Create(outfn)
				if err != nil {
					return err
				}
			}
		}
		err = enc.Reconstruct(shards, out)
		if err != nil {
			return err
		}

		for i := range out {
			if out[i] != nil {
				err := out[i].(*os.File).Close()
				if err != nil {
					return err
				}
			}
		}
		shards, size, err = openInput(datashards, rdushards, outfn)
		ok, err = enc.Verify(shards)
		if !ok {
			return err
		}
		if err != nil {
			return err
		}
	}

	f, err := os.Create(outfn)
	if err != nil {
		return err
	}

	shards, size, err = openInput(datashards, rdushards, outfn)
	if err != nil {
		return err
	}

	err = enc.Join(f, shards, int64(datashards)*size)
	return err
}

func reedSolomonRule(fsize int64) (int, int, error) {
	var count int64
	datachunk := int64(1)

	if fsize <= configs.SIZE_1KiB {
		if fsize <= 1 {
			return 1, 0, nil
		}
		datachunk = 2
		goto result
	}

	count = fsize / configs.SIZE_1GiB
	if count <= 1 {
		datachunk = 4
	} else {
		if count%2 == 0 {
			datachunk = count + 4
		} else {
			datachunk = count + 3
		}
	}

result:

	if datachunk > 20 {
		datachunk = 20
	}

	rdchunks := datachunk / 2

	if math.Ceil(float64(fsize)/float64(datachunk)*float64(datachunk+rdchunks)) > float64(fsize)*float64(1.5) {
		datachunk -= 1
	}
	return int(datachunk), int(datachunk / 2), nil
}

func openInput(dataShards, parShards int, fname string) (r []io.Reader, size int64, err error) {
	shards := make([]io.Reader, dataShards+parShards)
	for i := range shards {
		var infn string
		if i < 10 {
			infn = fmt.Sprintf("%s.00%d", fname, i)
		} else {
			infn = fmt.Sprintf("%s.0%d", fname, i)
		}
		f, err := os.Open(infn)
		if err != nil {
			shards[i] = nil
			continue
		} else {
			shards[i] = f
		}
		stat, err := f.Stat()
		if err != nil {
			return nil, 0, err
		}
		if stat.Size() > 0 {
			size = stat.Size()
		} else {
			shards[i] = nil
		}
	}
	return shards, size, nil
}
