package cmdline

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"scheduler-mining/configs"

	"gopkg.in/yaml.v2"
)

// command line parameters
func CmdlineInit() {
	var (
		err          error
		helpInfo     bool
		showVersion  bool
		confFilePath string
	)
	flag.BoolVar(&helpInfo, "h", false, "Print Help (this message) and exit")
	flag.BoolVar(&showVersion, "v", false, "Print version information and exit")
	flag.StringVar(&confFilePath, "c", "", "`configuration file`.")
	flag.Usage = usage
	flag.Parse()
	if helpInfo {
		flag.Usage()
		os.Exit(configs.Exit_Normal)
	}
	if showVersion {
		fmt.Println(configs.Version)
		os.Exit(configs.Exit_Normal)
	}
	if confFilePath == "" {
		fmt.Printf("\x1b[%dm[info]\x1b[0m Use '-h' to view help information\n", 43)
		os.Exit(configs.Exit_CmdLineParaErr)
	}
	_, err = os.Stat(confFilePath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file does not exist\n", 31, confFilePath)
		os.Exit(configs.Exit_ConfFileNotExist)
	}
	yamlFile, err := ioutil.ReadFile(confFilePath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file read error\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileTypeError)
	}
	err = yaml.Unmarshal(yamlFile, configs.Confile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileFormatError)
	}
}

func usage() {
	str := `CESS-Scheduler-Mining

Usage:
    `
	str += fmt.Sprintf("%v", os.Args[0])
	str += ` [Arguments] [file]

Arguments:
`
	fmt.Fprintf(os.Stdout, str)
	flag.PrintDefaults()
}
