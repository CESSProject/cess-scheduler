package fileauth

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"

	"github.com/corona10/goimagehash"
)

func GetImgSimHash() {
	var (
		err  error
		img1 image.Image
		img2 image.Image
	)
	file1, err := os.Open("12.jpeg")
	if err != nil {
		fmt.Printf("err1:%v\n", err)
		return
	}
	defer file1.Close()
	file2, err := os.Open("3.jpg")
	if err != nil {
		fmt.Printf("err2:%v\n", err)
		return
	}
	defer file2.Close()

	img1, err = jpeg.Decode(file1)
	if err != nil {
		fmt.Printf("err3:%v\n", err)
		return
	}
	img2, err = jpeg.Decode(file2)
	if err != nil {
		fmt.Printf("err4:%v\n", err)
		return
	}

	hash1, _ := goimagehash.AverageHash(img1)
	hash2, _ := goimagehash.AverageHash(img2)
	distance, _ := hash1.Distance(hash2)
	fmt.Printf("distance: %v\n", distance)

	hash1, _ = goimagehash.DifferenceHash(img1)
	hash2, _ = goimagehash.DifferenceHash(img2)
	distance, _ = hash1.Distance(hash2)
	fmt.Printf("distance: %v\n", distance)
	width1, height1 := 8, 8
	hash3, _ := goimagehash.ExtAverageHash(img1, width1, height1)
	hash4, _ := goimagehash.ExtAverageHash(img2, width1, height1)
	distance, _ = hash3.Distance(hash4)
	fmt.Printf("distance: %v\n", distance)
	// fmt.Printf("hash3 bit size: %v\n", hash3.Bits())
	// fmt.Printf("hash4 bit size: %v\n", hash4.Bits())

	width, height := 8, 8
	hash11, _ := goimagehash.ExtPerceptionHash(img1, width, height)
	hash12, _ := goimagehash.ExtPerceptionHash(img2, width, height)
	distance, _ = hash11.Distance(hash12)
	fmt.Printf("distance: %v\n", distance)

}
