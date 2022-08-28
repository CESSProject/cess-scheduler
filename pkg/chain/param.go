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

package chain

// cess chain state
const (
	State_FileBank    = "FileBank"
	State_FileMap     = "FileMap"
	State_Sminer      = "Sminer"
	State_SegmentBook = "SegmentBook"
	State_System      = "System"
)

// cess chain module method
const (
	// System
	System_Account = "Account"
	// Sminer
	Sminer_AllMinerItems  = "AllMiner"
	Sminer_MinerItems     = "MinerItems"
	Sminer_SegInfo        = "SegInfo"
	Sminer_PurchasedSpace = "PurchasedSpace"
	Sminer_TotalSpace     = "AvailableSpace"
	// FileMap
	FileMap_FileMetaInfo  = "File"
	FileMap_SchedulerInfo = "SchedulerMap"
	FileMap_SchedulerPuk  = "SchedulerPuk"
	// FileBank
	FileBank_UserSpaceList    = "UserSpaceList"
	FileBank_PurchasedPackage = "PurchasedPackage"
	FileBank_UserFilelist     = "UserHoldFileList"
	FileBank_FileRecovery     = "FileRecovery"
	// SegmentBook
	SegmentBook_UnVerifyProof = "UnVerifyProof"
)

// cess chain Transaction name
const (
	//
	Tx_FileBank_Update             = "FileBank.update"
	Tx_FileBank_Upload             = "FileBank.upload"
	Tx_FileBank_UploadFiller       = "FileBank.upload_filler"
	Tx_FileBank_ClearRecoveredFile = "FileBank.recover_file"
	//
	Tx_SegmentBook_VerifyProof = "SegmentBook.verify_proof"
	//
	Tx_FileMap_UpdateScheduler = "FileMap.update_scheduler"
	Tx_FileMap_Add_schedule    = "FileMap.registration_scheduler"
)

var (
	SSPrefix        = []byte{0x53, 0x53, 0x35, 0x38, 0x50, 0x52, 0x45}
	SubstratePrefix = []byte{0x2a}
	CessPrefix      = []byte{0x50, 0xac}
)
