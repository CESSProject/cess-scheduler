package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

const (
	letterIdBits = 6
	letterIdMask = 1<<letterIdBits - 1
	letterIdMax  = 63 / letterIdBits
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

// Bytes to Integer
func BytesToInteger(n []byte) (int32, error) {
	var x int32
	bytesBuffer := bytes.NewBuffer(n)
	err := binary.Read(bytesBuffer, binary.LittleEndian, &x)
	return x, err
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
	ctx1, _ := context.WithTimeout(context.Background(), 10*time.Second)
	output, err := exec.CommandContext(ctx1, "bash", "-c", "curl ifconfig.co").Output()
	if err == nil {
		result := strings.ReplaceAll(string(output), "\n", "")
		return strings.ReplaceAll(result, " ", ""), nil
	}

	ctx2, _ := context.WithTimeout(context.Background(), 10*time.Second)
	output, err = exec.CommandContext(ctx2, "bash", "-c", "curl cip.cc | grep  IP | awk '{print $3;}'").Output()
	if err == nil {
		result := strings.ReplaceAll(string(output), "\n", "")
		return strings.ReplaceAll(result, " ", ""), nil

	}
	ctx3, _ := context.WithTimeout(context.Background(), 10*time.Second)
	output, err = exec.CommandContext(ctx3, "bash", "-c", `curl ipinfo.io | grep \"ip\" | awk '{print $2;}'`).Output()
	if err == nil {
		result := strings.ReplaceAll(string(output), "\"", "")
		result = strings.ReplaceAll(result, ",", "")
		return strings.ReplaceAll(result, "\n", ""), nil
	}
	return "", errors.New("Please check your network status")
}

func Split(filefullpath string, blocksize, filesize int64) ([][]byte, uint64, error) {
	file, err := os.Open(filefullpath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	if filesize/blocksize == 0 {
		return nil, 0, errors.New("filesize invalid")
	}
	n := uint64(math.Ceil(float64(filesize / blocksize)))
	if n == 0 {
		n = 1
	}
	// matrix is indexed as m_ij, so the first dimension has n items and the second has s.
	matrix := make([][]byte, n)
	for i := uint64(0); i < n; i++ {
		piece := make([]byte, blocksize)
		_, err := file.Read(piece)
		if err != nil {
			return nil, 0, err
		}
		matrix[i] = piece
	}
	return matrix, n, nil
}

//
func RandStr(n int) string {
	src := rand.NewSource(time.Now().UnixNano())
	sb := strings.Builder{}
	sb.Grow(n)
	// A rand.Int63() generates 63 random bits, enough for letterIdMax letters!
	for i, cache, remain := n-1, src.Int63(), letterIdMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(baseStr) {
			sb.WriteByte(baseStr[idx])
			i--
		}
		cache >>= letterIdBits
		remain--
	}
	return sb.String()
}

func CalcHash(data []byte) (string, error) {
	if len(data) <= 0 {
		return "", errors.New("data is nil")
	}
	h := sha256.New()
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func GetStringWithoutNumbers(in string) string {
	var resu string
	resu = RemoveX(in, strconv.Itoa(0))
	for i := 1; i < 10; i++ {
		resu = RemoveX(resu, strconv.Itoa(i))
	}
	return resu
}

func RemoveX(str string, x string) string {
	var res string
	for i := 0; i < len(str); i++ {
		if string(str[i]) != x {
			res = res + string(str[i])
		}
	}
	return res
}
