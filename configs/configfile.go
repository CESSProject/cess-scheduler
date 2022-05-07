package configs

type SchedulerConfOnChain struct {
	CessChain     CessChain     `toml:"CessChain"`
	SchedulerInfo SchedulerInfo `toml:"SchedulerInfo"`
}

type CessChain struct {
	ChainAddr string `toml:"ChainAddr"`
}

type SchedulerInfo struct {
	ServiceAddr             string `toml:"ServiceAddr"`
	ServicePort             string `toml:"ServicePort"`
	DataDir                 string `toml:"DataDir"`
	ControllerAccountPhrase string `toml:"ControllerAccountPhrase"`
	StashAccountAddress     string `toml:"StashAccountAddress"`
}

var Confile = new(SchedulerConfOnChain)
var ConfigFilePath string

const DefaultConfigurationFileName = "conf_template.toml"
const ConfigFile_Templete = `[CessChain]
# CESS chain address
ChainAddr = ""

[SchedulerInfo]
# The IP address of the machine's public network used by the scheduler program.
ServiceAddr    = ""
# Port number monitored by the scheduler program
ServicePort    = ""
# Data storage directory
DataDir        = ""
# Phrase words or seeds for controller accounts.
ControllerAccountPhrase = ""
# Stash account address.
StashAccountAddress     = ""`
