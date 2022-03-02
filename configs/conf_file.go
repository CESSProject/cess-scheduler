package configs

type SchedulerConfOnChain struct {
	CessChain     CessChain     `toml:"CessChain"`
	SchedulerInfo SchedulerInfo `toml:"SchedulerInfo"`
}

type CessChain struct {
	ChainAddr string `toml:"ChainAddr"`
}

type SchedulerInfo struct {
	ServiceAddr    string `toml:"ServiceAddr"`
	ServicePort    string `toml:"ServicePort"`
	TransactionPrK string `toml:"TransactionPrK"`
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
# Phrase words or seeds for identity accounts.
TransactionPrK = ""`
