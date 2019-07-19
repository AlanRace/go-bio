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
	filename := "X:\\CRUK\\UnifiedWorkflowStudy\\AZ\\OI_from left_136,146.svs"

	tiffFile, err := Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer tiffFile.Close()

	ifdIndex := 0 //len(tiffFile.IFDList) - 2
	//ifdIndex = 5

	//fmt.Println(tiffFile)
	//tiffFile.IFDList[ifdIndex].PrintMetadata()

	fmt.Println(tiffFile.IFDList[ifdIndex].GetImageDimensions())
	//	fmt.Println(tiffFile.IFDList[ifdIndex].GetResolution())

	ifd := tiffFile.IFDList[ifdIndex]

	section := ifd.GetSectionAt(0, 0)

	data, err := section.GetRGBAData()

	img := image.NewRGBA(image.Rect(0, 0, int(section.sectionWidth), int(section.sectionHeight)))
	img.Pix = data

	f, err := os.Create("section.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, img)

	/*dataAccess := tiffFile.IFDList[ifdIndex].dataAccess

	tileAccess, ok := dataAccess.(*TileDataAccess)

	if ok {
		fmt.Println(tileAccess.GetTileDimensions())
		fmt.Println(tileAccess.GetTileGrid())

		fmt.Println(tileAccess.GetTileAt(0, 0))
		fmt.Println(tileAccess.GetTileAt(511, 511))
		fmt.Println(tileAccess.GetTileAt(512, 512))
		fmt.Println(tileAccess.GetTileAt(1000, 1000))
		fmt.Println(tileAccess.GetTileAt(48560, 33406))

		//fmt.Println(tiffFile.IFDList[ifdIndex].GetTileAt(tiffFile.IFDList[0].GetImageDimensions()))

		tile := tileAccess.GetSectionAt(0, 0)

		tileData, err := tileAccess.GetTileData(0, 0)
		if err != nil {
			log.Fatal(err)
		}

		var newData []uint8
		newData = make([]uint8, len(tileData)/3)

		for i := 0; i < len(newData); i++ {
			newData[i] = tileData[i*3]
		}

		fmt.Printf("tileData size: %d\n", len(tileData))

		tileWidth, tileLength := tileAccess.GetTileDimensions()
		img := image.NewGray(image.Rect(0, 0, int(tileWidth), int(tileLength)))
		img.Pix = newData

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

		img, err := stripAccess.GetImage()
		if err != nil {
			log.Fatal(err)
		}

		f, err := os.Create("strip.png")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		png.Encode(f, img)
	}
	*/
}
