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

//
const (
	LengthOfALine = 4096
	SIZE_1TB      = 1024 * 1024 * 1024 * 1024
	SIZE_1GB      = 1024 * 1024 * 1024
	SIZE_1MB      = 1024 * 1024
	SIZE_1KB      = 1024
	BlockSize     = 1024 * 1024
	ScanBlockSize = 512 * 1024
)

const (
	MinimumBalance = 2_000_000_000_000
	FillerSize     = 8 * 1024 * 1024
)

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

var (
	LogName = [9]string{"common", "upfile", "downfile", "filler", "panic", "vp", "smi", "sfm", "gf"}
)
