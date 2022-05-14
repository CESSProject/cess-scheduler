package configs

type Confile struct {
	RpcAddr     string `toml:"RpcAddr"`
	ServiceAddr string `toml:"ServiceAddr"`
	ServicePort string `toml:"ServicePort"`
	DataDir     string `toml:"DataDir"`
	CtrlPrk     string `toml:"CtrlPrk"`
	StashAcc    string `toml:"StashAcc"`
}

var C = new(Confile)
var ConfigFilePath string

const DefaultConfigurationFileName = "conf_template.toml"
const ConfigFile_Templete = `# The rpc address of the chain node
RpcAddr     = ""
# The IP address of the machine's public network used by the scheduler program
ServiceAddr = ""
# Port number monitored by the scheduler program
ServicePort = ""
# Data storage directory
DataDir     = ""
# Phrase or seed of the controller account
CtrlPrk     = ""
# The address of stash account
StashAcc    = ""`
