package fileauth

import "path/filepath"

func GetFileSimhash(filename string) (string, error) {
	fileSuffix := filepath.Ext(filename)
	switch fileSuffix {
	//text file
	case ".txt":
	case ".ini":
	case ".inf":
	case ".wtx":
	case ".xml":
	case ".json":
	case ".log":
	case ".cmd":
	case ".bat":
	case ".c":
	case ".cc":
	case ".cpp":
	case ".cs":
	case ".h":
	case ".go":
	case ".java":
	case ".htm":
	case ".html":
	case ".jsp":
	case ".js":
	case ".ts":
	case ".php":
	case ".rs":
	case ".py":
	//img file
	case ".jpg":
	case ".jpeg":
	case ".png":
	case ".bmp":
	case ".gif":
	default:
	}
	return "", nil
}
