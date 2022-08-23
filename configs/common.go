package configs

import "time"

// type and version
const Version = "cess-scheduler v0.5.1"

// return code
const (
	//success
	Code_200 = 200
	//not found
	Code_404 = 404
)

//
const (
	LengthOfALine = 4096
	SIZE_1TB      = 1024 * 1024 * 1024 * 1024
	SIZE_1GB      = 1024 * 1024 * 1024
	SIZE_1MB      = 1024 * 1024
	SIZE_1KB      = 1024
	BlockSize     = 1024 * 1024
	ScanBlockSize = 512 * 1024

	// the time to wait for the event, in seconds
	TimeToWaitEvents = time.Duration(time.Second * 15)
	BaseDir          = "scheduler"
)

var (
	PublicKey []byte
	//data dir
	LogFileDir    = "log"
	FileCacheDir  = "file"
	DbFileDir     = "db"
	SpaceCacheDir = "space"
)

const (
	HELP_common = `Please check with the following help information:
    1.Check if the wallet balance is sufficient
    2.Block hash:`
	HELP_register = `    3.Check the FileMap_RegistrationScheduler transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
	HELP_update = `    3.Check the FileMap_UpdateScheduler transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
)
