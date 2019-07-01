package qptiff

import (
	"fmt"
	"log"
	"testing"
	"os"
	"image"
	"image/png"
)

func TestLoad(t *testing.T) {
	filename := "C:\\Work\\PuffPiece\\kidney msi-if-imc_Scan1.qptiff"

	qptiffFile, err := Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer qptiffFile.Close()

	fmt.Println(qptiffFile)

	fmt.Println(qptiffFile.FilterMap["DAPI"])

	fullData := qptiffFile.Label.GetFullData()

	data := make([]byte, len(fullData)/3*4)

	for i := 0; i < len(data)/4; i++ {
		data[i*4] = fullData[i*3]
		data[i*4+1] = fullData[i*3+1]
		data[i*4+2] = fullData[i*3+2]
		data[i*4+3] = 255
	}

	width, length := qptiffFile.Label.GetImageDimensions()
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(length)))
	img.Pix = data

	f, err := os.Create("label.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, img)
}
