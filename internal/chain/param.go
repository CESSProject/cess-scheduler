package chain

// cess chain state
const (
	State_System      = "System"
	State_Sminer      = "Sminer"
	State_SegmentBook = "SegmentBook"
	State_FileBank    = "FileBank"
	State_FileMap     = "FileMap"
)

// cess chain module method
const (
	System_Account            = "Account"
	Sminer_AllMinerItems      = "AllMiner"
	Sminer_MinerItems         = "MinerItems"
	Sminer_SegInfo            = "SegInfo"
	FileMap_FileMetaInfo      = "File"
	FileMap_SchedulerInfo     = "SchedulerMap"
	FileBank_UserSpaceList    = "UserSpaceList"
	FileBank_PurchasedPackage = "PurchasedPackage"
	FileBank_UserFilelist     = "UserHoldFileList"
	Sminer_PurchasedSpace     = "PurchasedSpace"
	Sminer_TotalSpace         = "AvailableSpace"
	FileMap_SchedulerPuk      = "SchedulerPuk"
	SegmentBook_UnVerifyProof = "UnVerifyProof"
	FileBank_FileRecovery     = "FileRecovery"
)

// cess chain Transaction name
const (
	ChainTx_FileBank_Update       = "FileBank.update"
	ChainTx_FileMap_Add_schedule  = "FileMap.registration_scheduler"
	Tx_FileBank_Upload            = "FileBank.upload"
	ChainTx_FileBank_UploadFiller = "FileBank.upload_filler"
	SegmentBook_VerifyProof       = "SegmentBook.verify_proof"
	FileBank_ClearRecoveredFile   = "FileBank.recover_file"
	FileMap_UpdateScheduler       = "FileMap.update_scheduler"
)

var (
	SSPrefix        = []byte{0x53, 0x53, 0x35, 0x38, 0x50, 0x52, 0x45}
	SubstratePrefix = []byte{0x2a}
	CessPrefix      = []byte{0x50, 0xac}
)
