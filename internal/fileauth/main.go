package fileauth

import (
	"fmt"
	"os"
	"scheduler-mining/configs"
	"scheduler-mining/internal/logger"
	"strings"

	"github.com/pkg/errors"
)

func FileAuth_Init() {
	_, err := os.Stat(dictPath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please download our dictionary package and use it according to the instructions. %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(configs.Exit_Normal)
	}
}

func GetFileSimhash(filetype, filepath string) (string, string, error) {
	var (
		err      error
		hashtype string
		simhash  string
	)
	switch strings.ToLower(filetype) {
	//text file
	case ".txt", ".ini", ".inf", ".wtx", ".xml", ".json", ".log", ".cmd", ".bat",
		".c", ".cc", ".cpp", ".cs", ".h", ".go", ".java", ".htm", ".html", ".jsp",
		".js", ".ts", ".php", ".rs", ".py":
		simhash, err = getTextSimhash(filepath)
		if err != nil {
			return "", hashtype, errors.Wrap(err, "getTextSimhash err")
		}
		hashtype = "text"
	//img file
	case ".jpg", ".jpeg", ".png", ".bmp", ".gif", ".tiff":
		simhash, err = getImgSimHash(filetype, filepath)
		if err != nil {
			return "", hashtype, errors.Wrap(err, "getTextSimhash err")
		}
		hashtype = "image"
	default:
		return "", hashtype, errors.New("Unsupported file format")
	}
	return simhash, hashtype, nil
}
