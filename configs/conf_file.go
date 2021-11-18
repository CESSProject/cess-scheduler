package configs

type MinerOnChain struct {
	CessChain CessChain `json:"cessChain"`
	MinerData MinerData `json:"minerData"`
}

type CessChain struct {
	RpcAddr string `json:"rpcAddr"`
}

type MinerData struct {
	ServiceIpAddress            string `json:"serviceIpAddress"`
	ServicePort                 string `json:"servicePort"`
	IdentifyAccountPhraseOrSeed string `json:"identifyAccountPhraseOrSeed"`
}

var Confile = new(MinerOnChain)
