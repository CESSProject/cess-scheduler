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

package console

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/configfile"
	"github.com/btcsuite/btcutil/base58"
	"github.com/spf13/cobra"
)

// updateCmd is used to update the communication address
//
// Usage:
//
//	scheduler update <ipv4:port> or <domain name>
func updateCmd(cmd *cobra.Command, args []string) {
	if len(os.Args) >= 3 {
		addr := strings.Split(os.Args[2], ":")
		if len(addr) == 2 {
			if !isIPv4(addr[0]) {
				if !strings.Contains(addr[0], "http") &&
					!strings.Contains(addr[0], "https") &&
					!strings.Contains(addr[0], "www") {
					log.Println("Please enter <ipv4:port> or <domain name>")
					os.Exit(1)
				}
			}
			port, err := strconv.Atoi(addr[1])
			if err != nil {
				log.Println("Invalid port number")
				os.Exit(1)
			}
			if port < 1025 || port > 65535 {
				log.Println("The port number range is 1024~65535")
				os.Exit(1)
			}
		}

		if !strings.Contains(os.Args[2], "http") &&
			!strings.Contains(os.Args[2], "https") &&
			!strings.Contains(os.Args[2], "www") {
			log.Println("Please enter <ipv4:port> or <domain name>")
			os.Exit(1)
		}

		// config file
		var configFilePath string
		configpath1, _ := cmd.Flags().GetString("config")
		configpath2, _ := cmd.Flags().GetString("c")
		if configpath1 != "" {
			configFilePath = configpath1
		} else {
			configFilePath = configpath2
		}

		confile := configfile.NewConfigfile()
		if err := confile.Parse(configFilePath); err != nil {
			log.Println(err)
			os.Exit(1)
		}

		// chain client
		c, err := chain.NewChainClient(
			confile.GetRpcAddr(),
			confile.GetCtrlPrk(),
			time.Duration(time.Second*15),
		)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}

		txhash, err := c.Update(base58.Encode([]byte(os.Args[2])))
		if err != nil {
			if err.Error() == chain.ERR_RPC_EMPTY_VALUE.Error() {
				log.Println("[err] Please check your wallet balance.")
			} else {
				if txhash != "" {
					msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
					msg += configs.HELP_update
					log.Printf("[pending] %v\n", msg)
				} else {
					log.Printf("[err] %v\n", err)
				}
			}
			os.Exit(1)
		}
		log.Println("[ok] success")
		os.Exit(0)
	}
	log.Println("[err] Please enter <ipv4:port>")
	os.Exit(1)
}

func isIPv4(ipAddr string) bool {
	ip := net.ParseIP(ipAddr)
	return ip != nil && strings.Contains(ipAddr, ".")
}
