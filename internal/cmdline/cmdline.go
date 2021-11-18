package cmdline

import (
	"flag"
	"fmt"
	"os"
	"scheduler-mining/configs"

	"github.com/spf13/viper"
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
		fmt.Printf("\x1b[%dmUse '-h' to view help information.\x1b[0m\n", 32)
		os.Exit(configs.Exit_CmdLineParaErr)
	}
	_, err = os.Stat(confFilePath)
	if err != nil {
		fmt.Printf("\x1b[%dmErr:The '%v' file does not exist.\x1b[0m\n", 31, confFilePath)
		os.Exit(configs.Exit_ConfFileNotExist)
	}
	viper.SetConfigFile(confFilePath)
	viper.SetConfigType("toml")
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("\x1b[%dmErr:The '%v' file type error.\x1b[0m\n", 31, confFilePath)
		os.Exit(configs.Exit_ConfFileTypeError)
	}
	err = viper.Unmarshal(configs.Confile)
	if err != nil {
		fmt.Printf("\x1b[%dmErr:The '%v' file format error.\x1b[0m\n", 31, confFilePath)
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
