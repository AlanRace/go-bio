package qptiff

import (
	"fmt"
	"log"
	"testing"
	"os"
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

	/*fullData, err := qptiffFile.Label.GetFullData()
	if err != nil {
		log.Fatal(err)
	}

	data := make([]byte, len(fullData)/3*4)

	for i := 0; i < len(data)/4; i++ {
		data[i*4] = fullData[i*3]
		data[i*4+1] = fullData[i*3+1]
		data[i*4+2] = fullData[i*3+2]
		data[i*4+3] = 255
	}

	qptiffFile.Label.PrintMetadata()

	width, length := qptiffFile.Label.GetImageDimensions()
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(length)))
	img.Pix = data*/

	dapi := qptiffFile.FilterMap["DAPI"]
	
	for _, ifd := range dapi.IFDList {
		fmt.Println(ifd.GetImageDimensions())
		fmt.Println(ifd.GetResolution())
	}

	img, err := qptiffFile.Label.GetImage()
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create("label.png")
	if err != nil {
		panic(err)
	}
	png.Encode(f, img)
	f.Close()

	img, err = qptiffFile.Overview.GetImage()
	if err != nil {
		log.Fatal(err)
	}

	f, err = os.Create("overview.png")
	if err != nil {
		panic(err)
	}
	png.Encode(f, img)
	f.Close()

	img, err = qptiffFile.Thumbnail.GetImage()
	if err != nil {
		log.Fatal(err)
	}

	f, err = os.Create("thumbnail.png")
	if err != nil {
		panic(err)
	}
	png.Encode(f, img)
	f.Close()
}
