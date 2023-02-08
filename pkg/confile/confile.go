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

package confile

import (
	"os"
	"path"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	DefaultConfigurationFile      = "./conf.toml"
	ConfigurationFileTemplateName = "conf_template.toml"
	ConfigurationFileTemplete     = `# The rpc address of the chain node
RpcAddr     = ""
# The IP address of the machine's public network
ServiceAddr = ""
# Port number monitored by the scheduler program
ServicePort = 15000
# Sgx service communication port
SgxPort     = 80
# Data storage directory
DataDir     = ""
# Phrase or seed of the controller account
CtrlPrk     = ""
# The address of stash account
StashAcc    = ""`
)

type Confiler interface {
	Parse(path string) error
	GetRpcAddr() string
	GetServiceAddr() string
	GetServicePort() uint32
	GetSgxPort() uint32
	GetDataDir() string
	GetCtrlPrk() string
	GetStashAcc() string
}

type confile struct {
	RpcAddr     string `name:"RpcAddr" toml:"RpcAddr" yaml:"RpcAddr"`
	ServiceAddr string `name:"ServiceAddr" toml:"ServiceAddr" yaml:"ServiceAddr"`
	ServicePort uint32 `name:"ServicePort" toml:"ServicePort" yaml:"ServicePort"`
	SgxPort     uint32 `name:"SgxPort" toml:"SgxPort" yaml:"SgxPort"`
	DataDir     string `name:"DataDir" toml:"DataDir" yaml:"DataDir"`
	CtrlPrk     string `name:"CtrlPrk" toml:"CtrlPrk" yaml:"CtrlPrk"`
	StashAcc    string `name:"StashAcc" toml:"StashAcc" yaml:"StashAcc"`
}

func NewConfigfile() Confiler {
	return &confile{}
}

func (c *confile) Parse(fpath string) error {
	var confilePath = fpath
	if confilePath == "" {
		confilePath = DefaultConfigurationFile
	}
	fstat, err := os.Stat(confilePath)
	if err != nil {
		return errors.Errorf("Parse: %v", err)
	}
	if fstat.IsDir() {
		return errors.Errorf("The '%v' is not a file", confilePath)
	}

	viper.SetConfigFile(confilePath)
	viper.SetConfigType(path.Ext(confilePath)[1:])

	err = viper.ReadInConfig()
	if err != nil {
		return errors.Errorf("ReadInConfig: %v", err)
	}
	err = viper.Unmarshal(c)
	if err != nil {
		return errors.Errorf("Unmarshal: %v", err)
	}

	_, err = signature.KeyringPairFromSecret(c.CtrlPrk, 0)
	if err != nil {
		return errors.Errorf("Secret: %v", err)
	}

	_, err = utils.DecodePublicKeyOfCessAccount(c.StashAcc)
	if err != nil {
		return errors.Errorf("Decode: %v", err)
	}

	if c.DataDir == "" ||
		c.RpcAddr == "" ||
		c.ServiceAddr == "" {
		return errors.New("The configuration file cannot have empty entries")
	}

	if c.ServicePort < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", c.ServicePort)
	}
	if c.ServicePort > 65535 {
		return errors.New("The port number cannot exceed 65535")
	}

	if c.SgxPort == 0 || c.SgxPort > 65535 {
		return errors.New("Illegal sgx service port")
	}

	fstat, err = os.Stat(c.DataDir)
	if err != nil {
		err = os.MkdirAll(c.DataDir, configs.DirPermission)
		if err != nil {
			return err
		}
	} else {
		if !fstat.IsDir() {
			return errors.Errorf("The '%v' is not a directory", c.DataDir)
		}
	}
	return nil
}

func (c *confile) GetRpcAddr() string {
	return c.RpcAddr
}

func (c *confile) GetServiceAddr() string {
	return c.ServiceAddr
}

func (c *confile) GetServicePort() uint32 {
	return c.ServicePort
}

func (c *confile) GetSgxPort() uint32 {
	return c.SgxPort
}

func (c *confile) GetDataDir() string {
	return c.DataDir
}

func (c *confile) GetCtrlPrk() string {
	return c.CtrlPrk
}

func (c *confile) GetStashAcc() string {
	return c.StashAcc
}
