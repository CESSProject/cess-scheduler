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

package configfile

import (
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
	GetRpcAddr() string
	GetServiceAddr() string
	GetServicePort() string
	GetDataDir() string
	GetCtrlPrk() string
	GetStashAcc() string
}

type configfile struct {
	RpcAddr     string `name:"RpcAddr" toml:"RpcAddr" yaml:"RpcAddr"`
	ServiceAddr string `name:"ServiceAddr" toml:"ServiceAddr" yaml:"ServiceAddr"`
	ServicePort string `name:"ServicePort" toml:"ServicePort" yaml:"ServicePort"`
	DataDir     string `name:"DataDir" toml:"DataDir" yaml:"DataDir"`
	CtrlPrk     string `name:"CtrlPrk" toml:"CtrlPrk" yaml:"CtrlPrk"`
	StashAcc    string `name:"StashAcc" toml:"StashAcc" yaml:"StashAcc"`
}

func NewConfigfile() Configfiler {
	return &configfile{}
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
	err = viper.Unmarshal(c)
	if err != nil {
		return errors.Errorf("Unmarshal: %v", err)
	}

	if c.CtrlPrk == "" ||
		c.DataDir == "" ||
		c.RpcAddr == "" ||
		c.ServiceAddr == "" ||
		c.StashAcc == "" ||
		c.ServicePort == "" {
		return errors.New("The configuration file cannot have empty entries")
	}

	port, err := strconv.Atoi(c.ServicePort)
	if err != nil {
		return errors.New("The port number should be between 1025~65535")
	}
	if port < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", port)
	}
	if port > 65535 {
		return errors.New("The port number cannot exceed 65535")
	}

	fstat, err = os.Stat(c.DataDir)
	if err != nil {
		err = os.MkdirAll(c.DataDir, os.ModeDir)
		if err != nil {
			return err
		}
	}

	if !fstat.IsDir() {
		return errors.Errorf("The '%v' is not a directory", c.DataDir)
	}

	return nil
}

func (c *configfile) GetRpcAddr() string {
	return c.RpcAddr
}

func (c *configfile) GetServiceAddr() string {
	return c.ServiceAddr
}

func (c *configfile) GetServicePort() string {
	return c.ServicePort
}

func (c *configfile) GetDataDir() string {
	return c.DataDir
}

func (c *configfile) GetCtrlPrk() string {
	return c.CtrlPrk
}

func (c *configfile) GetStashAcc() string {
	return c.StashAcc
}
