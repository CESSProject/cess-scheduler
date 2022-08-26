package configs

import (
	"cess-scheduler/tools"
	"os"
	"path"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	DefaultConfigurationFile      = "./conf.toml"
	ConfigurationFileTemplateName = "conf_template.toml"
	ConfigurationFileTemplete     = `# The rpc address of the chain node
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
)

type Configfiler interface {
	Parse(path string) error
}

type Confile struct {
	RpcAddr     string `name:"RpcAddr" toml:"RpcAddr" yaml:"RpcAddr"`
	ServiceAddr string `name:"ServiceAddr" toml:"ServiceAddr" yaml:"ServiceAddr"`
	ServicePort string `name:"ServicePort" toml:"ServicePort" yaml:"ServicePort"`
	DataDir     string `name:"DataDir" toml:"DataDir" yaml:"DataDir"`
	CtrlPrk     string `name:"CtrlPrk" toml:"CtrlPrk" yaml:"CtrlPrk"`
	StashAcc    string `name:"StashAcc" toml:"StashAcc" yaml:"StashAcc"`
}

type configfile struct {
	confile *Confile
}

func NewConfigfile(confile *Confile) Configfiler {
	return &configfile{
		confile: confile,
	}
}

func (c *configfile) Parse(fpath string) error {
	var configFilePath = fpath
	if configFilePath == "" {
		configFilePath = DefaultConfigurationFile
	}
	fstat, err := os.Stat(configFilePath)
	if err != nil {
		return errors.Errorf("Parse: %v", err)
	}
	if fstat.IsDir() {
		return errors.Errorf("The '%v' is not a file", configFilePath)
	}

	viper.SetConfigFile(configFilePath)
	viper.SetConfigType(path.Ext(configFilePath)[1:])

	err = viper.ReadInConfig()
	if err != nil {
		return errors.Errorf("ReadInConfig: %v", err)
	}
	err = viper.Unmarshal(c.confile)
	if err != nil {
		return errors.Errorf("Unmarshal: %v", err)
	}

	if c.confile.CtrlPrk == "" ||
		c.confile.DataDir == "" ||
		c.confile.RpcAddr == "" ||
		c.confile.ServiceAddr == "" ||
		c.confile.StashAcc == "" ||
		c.confile.ServicePort == "" {
		return errors.New("The configuration file cannot have empty entries")
	}

	port, err := strconv.Atoi(c.confile.ServicePort)
	if err != nil {
		return errors.New("The port number should be between 1025~65535")
	}
	if port < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", port)
	}
	if port > 65535 {
		return errors.New("The port number cannot exceed 65535")
	}

	err = tools.CreatDirIfNotExist(c.confile.DataDir)
	if err != nil {
		return errors.Errorf("Creat dir: %v", err)
	}

	//
	// configs.PublicKey, err = chain.GetPublicKeyByPrk(c.confile.CtrlPrk)
	// if err != nil {
	// 	log.Printf("[err] %v\n", err)
	// 	os.Exit(1)
	// }
	return nil
}
