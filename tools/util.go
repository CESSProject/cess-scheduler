package tools

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

// Determine if the operating system is linux
func RunOnLinuxSystem() bool {
	return runtime.GOOS == "linux"
}

// Allocate all cores to the program
func SetAllCores() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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

// Calculate the file hash value
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

// Calculate the file hash value
func CalcFileHash2(f *os.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Get a random integer in a specified range
func RandomInRange(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

// Write string content to file
func WriteStringtoFile(content, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

//  ----------------------- Random key -----------------------
const baseStr = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()[]{}+-*/_=."

// Generate random password
func GetRandomkey(length uint8) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	bytes := make([]byte, length)
	l := len(baseStr)
	for i := uint8(0); i < length; i++ {
		bytes[i] = baseStr[r.Intn(l)]
	}
	return string(bytes)
}

// bytes to string
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// string to bytes
func S2B(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// Create a directory
func CreatDirIfNotExist(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		return os.MkdirAll(dir, os.ModeDir)
	}
	return nil
}

// Get the name of a first-level subdirectory in a given directory
func Post(url string, para interface{}) ([]byte, error) {
	body, err := json.Marshal(para)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	var resp = new(http.Response)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return respBody, err
	}
	return nil, err
}

// Get external network ip
func GetExternalIp() (string, error) {
	output, err := exec.Command("bash", "-c", "curl ifconfig.co").Output()
	// output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	result := strings.Replace(string(output), "\n", "", -1)
	return strings.Replace(result, " ", "", -1), nil
}
