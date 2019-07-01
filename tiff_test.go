package tiff

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	filename := "C:\\Work\\PuffPiece\\kidney msi-if-imc_Scan1.qptiff"

	tiffFile, err := Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer tiffFile.Close()

	ifdIndex := len(tiffFile.IFDList) - 2
	ifdIndex = 5

	//fmt.Println(tiffFile)
	tiffFile.IFDList[ifdIndex].PrintMetadata()

	fmt.Println(tiffFile.IFDList[ifdIndex].GetImageDimensions())

	dataAccess := tiffFile.IFDList[ifdIndex].dataAccess

	tileAccess, ok := dataAccess.(*TileDataAccess)

	if ok {
		fmt.Println(tileAccess.GetTileDimensions())
		fmt.Println(tileAccess.GetTileGrid())

		fmt.Println(tileAccess.GetTileAt(0, 0))
		fmt.Println(tileAccess.GetTileAt(511, 511))
		fmt.Println(tileAccess.GetTileAt(512, 512))
		fmt.Println(tileAccess.GetTileAt(1000, 1000))

		//fmt.Println(tiffFile.IFDList[ifdIndex].GetTileAt(tiffFile.IFDList[0].GetImageDimensions()))

		tileData, err := tileAccess.GetTileData(5, 5)
		if err != nil {
			log.Fatal(err)
		}

		tileWidth, tileLength := tileAccess.GetTileDimensions()
		img := image.NewGray(image.Rect(0, 0, int(tileWidth), int(tileLength)))
		img.Pix = tileData

		f, err := os.Create("tile.png")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		png.Encode(f, img)
	}

	stripAccess, ok := dataAccess.(*StripDataAccess)

	if ok {
		fmt.Println(stripAccess.GetStripDimensions())

		stripData, err := stripAccess.GetData(stripAccess.GetDataIndexAt(180, 180))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(len(stripData))

		fullData, err := stripAccess.GetFullData()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(len(fullData))

		data := make([]byte, len(fullData)/3*4)

		for i := 0; i < len(data)/4; i++ {
			data[i*4] = fullData[i*3]
			data[i*4+1] = fullData[i*3+1]
			data[i*4+2] = fullData[i*3+2]
			data[i*4+3] = 255
		}

		//stripWidth, stripLength := stripAccess.GetStripDimensions()
		width, length := stripAccess.GetImageDimensions()
		img := image.NewRGBA(image.Rect(0, 0, int(width), int(length)))
		img.Pix = data

		f, err := os.Create("strip.png")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		png.Encode(f, img)
	}
}
