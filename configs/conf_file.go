package configs

type SchedulerConfOnChain struct {
	CessChain     CessChain     `yaml:"cessChain"`
	SchedulerData SchedulerData `yaml:"schedulerData"`
}

type CessChain struct {
	RpcAddr string `yaml:"rpcAddr"`
}

type SchedulerData struct {
	ServiceIpAddr         string `yaml:"serviceIpAddress"`
	ServicePort           string `yaml:"servicePort"`
	IncomeAccountPubkey   string `yaml:"incomeAccountPubkey"`
	IdAccountPhraseOrSeed string `yaml:"idAccountPhraseOrSeed"`
}

var (
	Cd0url              string
	InitialClusterState string
)

var Confile = new(SchedulerConfOnChain)
var ConfigFilePath string

const DefaultConfigurationFileName = "conf_template.toml"
const ConfigFile_Templete = `[cessChain]
# CESS chain address
rpcAddr = "ws://106.15.44.155:9949/"

[schedulerData]
# The IP address of the machine's public network used by the mining program.
serviceIpAddress            = ""
# Port number monitored by the mining program
servicePort                 = ""
# Public key of income account.
incomeAccountPubkey    = ""
# Phrase words or seeds for identity accounts.
idAccountPhraseOrSeed  = ""`
