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

package configs

// account
const (
	// CESS token precision
	CESSTokenPrecision = 1_000_000_000_000
	// MinimumBalance is the minimum balance required for the program to run
	// The unit is pico
	MinimumBalance = 2 * CESSTokenPrecision
)

// byte size
const (
	SIZE_1KiB = 1024
	SIZE_1MiB = 1024 * SIZE_1KiB
	SIZE_1GiB = 1024 * SIZE_1MiB
)

// filler
const (
	// FillerSize is the fill size of the uncommissioned data segment
	FillerSize = 8 * SIZE_1MiB
	// FillerLineLength is the number of characters in a line
	FillerLineLength = 4096
	// BlockSize is the block size when pbc is calculated
	BlockSize = SIZE_1MiB
	// ScanBlockSize is the size of the scan and cannot be larger than BlockSize
	ScanBlockSize = BlockSize / 2
)

const (
	Max_Miner_Connected = 3
	Num_Filler_Reserved = 5
	Max_Filler_Meta     = 100
)

// explanation
const (
	HELP_common = `Please check with the following help information:
    1.Check if the wallet balance is sufficient
    2.Block hash:`
	HELP_register = `    3.Check the FileMap.RegistrationScheduler transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
	HELP_update = `    3.Check the FileMap.UpdateScheduler transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
)

// log
var (
	LogName = [9]string{"common", "upfile", "downfile", "filler", "panic", "vp", "smi", "sfm", "gf"}
)
