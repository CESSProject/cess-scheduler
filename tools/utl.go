package tools

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
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
	_, err := net.DialTimeout("tcp", address, 3*time.Second)
	return err == nil
}

// Integer to bytes
func IntegerToBytes(n interface{}) ([]byte, error) {
	bytesBuffer := bytes.NewBuffer([]byte{})
	t := reflect.TypeOf(n)
	switch t.Kind() {
	case reflect.Int16:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint16:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Int:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Int32:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint32:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Int64:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint64:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	default:
		return nil, errors.New("unsupported type")
	}
}

// Bytes to Integer
func BytesToInteger(n []byte) (int32, error) {
	var x int32
	bytesBuffer := bytes.NewBuffer(n)
	err := binary.Read(bytesBuffer, binary.LittleEndian, &x)
	return x, err
}

func Uint32ToIp(n uint32) string {
	ip := fmt.Sprintf("%v", uint8(n>>24))
	ip += "."
	ip += fmt.Sprintf("%v", uint8(n>>16))
	ip += "."
	ip += fmt.Sprintf("%v", uint8(n>>8))
	ip += "."
	ip += fmt.Sprintf("%v", uint8(n))
	return ip
}

func CalcFileHash(fpath string) (string, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func CleanLocalRecord(filename string) {
	path, _ := os.Getwd()
	filepath.Walk(path, func(path string, fi os.FileInfo, err error) error {
		if nil == fi {
			return err
		}
		if !fi.IsDir() {
			return nil
		}
		fname := fi.Name()
		if strings.Contains(fname, filename) {
			err := os.RemoveAll(path)
			if err != nil {
				fmt.Println("Delete dir error:", err)
			}
		}
		return nil
	})
}

func RandomInRange(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func WriteStringtoFile(content, fileName string) error {
	var (
		err  error
		name string
		//filesuffix string
		//fileprefix string
	)
	name = fileName
	// _, err = os.Stat(name)
	// if err == nil {
	// 	filesuffix = filepath.Ext(name)
	// 	fileprefix = name[0 : len(name)-len(filesuffix)]
	// 	fileprefix = fileprefix + fmt.Sprintf("_%v", strconv.FormatInt(time.Now().UnixNano(), 10))
	// 	name = fileprefix + filesuffix
	// }
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return errors.Wrap(err, "OpenFile err")
	}
	defer f.Close()
	_, err = f.Write([]byte(content))
	if err != nil {
		return errors.Wrap(err, "f.Write err")
	}
	return nil
}
