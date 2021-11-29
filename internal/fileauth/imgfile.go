package fileauth

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/corona10/goimagehash"
	"github.com/pkg/errors"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

func getImgSimHash(imgtype, imgpath string) (string, error) {
	var (
		err error
		img image.Image
	)
	imgfile, err := os.Open(imgpath)
	if err != nil {
		return "", errors.Wrap(err, "os.Open err")
	}
	defer imgfile.Close()
	switch imgtype {
	case ".jpg", "jpeg":
		img, err = jpeg.Decode(imgfile)
		if err != nil {
			return "", errors.Wrap(err, "jpeg.Decode err")
		}
	case ".png":
		img, err = png.Decode(imgfile)
		if err != nil {
			return "", errors.Wrap(err, "png.Decode err")
		}

	case ".gif":
		img, err = gif.Decode(imgfile)
		if err != nil {
			return "", errors.Wrap(err, "gif.Decode err")
		}

	case ".bmp":
		img, err = bmp.Decode(imgfile)
		if err != nil {
			return "", errors.Wrap(err, "bmp.Decode err")
		}

	case ".tiff":
		img, err = tiff.Decode(imgfile)
		if err != nil {
			return "", errors.Wrap(err, "tiff.Decode err")
		}
	default:
		return "", errors.New("unsupport img type")
	}
	averHash, err := goimagehash.AverageHash(img)
	if err != nil {
		return "", errors.Wrap(err, "ExtAverageHash err")
	}

	return fmt.Sprintf("%v", averHash.GetHash()), nil
}
