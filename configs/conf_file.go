package configs

type MinerOnChain struct {
	CessChain      CessChain      `yaml:"cessChain"`
	MinerData      MinerData      `yaml:"minerData"`
	RotationModule RotationModule `yaml:"rotationmodule"`
}

type CessChain struct {
	RpcAddr string `yaml:"rpcAddr"`
}

type MinerData struct {
	ServiceIpAddr         string `yaml:"serviceIpAddr"`
	ServicePort           string `yaml:"servicePort"`
	IdAccountPhraseOrSeed string `yaml:"idAccountPhraseOrSeed"`
}

type RotationModule struct {
	Endpoint                 string `yaml:"endpoints"`
	Name                     string `yaml:"name"`
	InitialAdvertisePeerUrls string `yaml:"initialAdvertisePeerUrls"`
	ListenPeerUrls           string `yaml:"listenPeerUrls"`
	ListenClientUrls         string `yaml:"listenClientUrls"`
	AdvertiseClientUrls      string `yaml:"advertiseClientUrls"`
	InitialClusterToken      string `yaml:"initialClusterToken"`
	InitialClusterState      string `yaml:"initialClusterState"`
	Username                 string `yaml:"user"`
	Password                 string `yaml:"password"`
	Cd0url                   string `yaml:"cd0url"`
	RootTotal                int    `yaml:"rootTotal"`
	NormalTotal              int64  `yaml:"normalTotal"`
	PollingTime              int64  `yaml:"PollingTime"`
}

var (
	Cd0url              string
	InitialClusterState string
)

var Confile = new(MinerOnChain)
