package tools

import (
	"errors"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func RunOnLinuxSystem() bool {
	return runtime.GOOS == "linux"
}

func RunWithRootPrivileges() bool {
	return os.Geteuid() == 0
}

func SetAllCores() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

// parse ip
func ParseIpPort(ip string) (string, string, error) {
	if ip != "" {
		ip_port := strings.Split(ip, ":")
		if len(ip_port) == 1 {
			isipv4 := net.ParseIP(ip_port[0])
			if isipv4 != nil {
				return ip + ":15001", ":15001", nil
			}
			return ip_port[0], ":15001", nil
		}
		if len(ip_port) == 2 {
			_, err := strconv.ParseUint(ip_port[1], 10, 16)
			if err != nil {
				return "", "", err
			}
			return ip, ":" + ip_port[1], nil
		}
		return "", "", errors.New(" The IP address is incorrect")
	} else {
		return "", "", errors.New(" The IP address is nil")
	}
}

//Judge whether IP can connect with TCP normally.
//Returning true means normal.
func TestConnectionWithTcp(ip string) bool {
	if ip == "" {
		return false
	}
	tmp := strings.Split(ip, ":")
	address := ""
	if len(tmp) > 1 {
		address = ip
	} else if len(tmp) == 1 {
		address = net.JoinHostPort(ip, "80")
	} else {
		return false
	}
	_, err := net.DialTimeout("tcp", address, 2*time.Second)
	return err == nil
}
