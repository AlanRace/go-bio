package gobio

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	//filename := "X:\\CRUK\\UnifiedWorkflowStudy\\AZ\\OI_from left_136,146.svs"
	//filename := "C:\\Work\\PuffPiece\\20191128_3Dunified_blueset_set7_orbisims_307_downsampled3x.tif"
	//filename := "C:\\Work\\Registration\\20151030_PDAC_set2_neg_80um_s66_Ivis_Lab_10x.tif"
	//filename := "C:\\Work\\AZ\\Stephanie\\20190913_CKD_Adenosine_CXCR2_DESIpos_30um_slide19_Section 09\\meanImage_cluster48.tif"
	//filename := "C:\\Work\\AZ\\Stephanie\\S.Ling H&E 5_09_b.tif"
	filename := "D:\\AZ\\Gemcitabine\\GEMTAB_Scan1.qptiff"

	tiffFile, err := Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer tiffFile.Close()

	ifdIndex := 0 //len(tiffFile.IFDList) - 2
	ifdIndex = 20

	//fmt.Println(tiffFile)
	//tiffFile.IFDList[ifdIndex].PrintMetadata()

	fmt.Println(tiffFile.IFDList[ifdIndex].GetImageDimensions())
	fmt.Printf("IsReducedResolutionImage? %v\n", tiffFile.IFDList[ifdIndex].IsReducedResolutionImage())
	compression, _ := tiffFile.IFDList[ifdIndex].GetCompression()
	fmt.Printf("Compression? %v\n", compression)
	//	fmt.Println(tiffFile.IFDList[ifdIndex].GetResolution())

	ifd := tiffFile.IFDList[ifdIndex]

	pi, err := ifd.GetPhotometricInterpretation()
	if err != nil {
		panic(err)
	}

	fmt.Println(pi.String())

	section := ifd.GetSectionAt(0, 0)
	data, err := section.GetRGBAData()
	if err != nil {
		panic(err)
	}

	img := image.NewRGBA(image.Rect(0, 0, int(section.Width), int(section.Height)))
	img.Pix = data

	f, err := os.Create("section.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, img)

	// Check can output whole file
	//	data = nil

	width, height := ifd.GetImageDimensions()

	img = image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	gridX, gridY := ifd.GetSectionGrid()
	fmt.Printf("Found grid dimensions: %d x %d (%d x %d)\n", gridX, gridY, width, height)

	curY := 0
	for y := uint32(0); y < gridY; y++ {
		curX := 0
		for x := uint32(0); x < gridX; x++ {
			section := ifd.GetSection(y*gridX + x)

			fmt.Printf("Found section dimensions (%d, %d): %d x %d\n", x, y, int(section.Width), int(section.Height))
			sectionImage := image.NewRGBA(image.Rect(0, 0, int(section.Width), int(section.Height)))

			rgbaData, err := section.GetRGBAData()
			if err != nil {
				log.Println(err)
				continue
			}

			if len(rgbaData) < int(section.Width)*int(section.Height)*4 {
				fmt.Printf("[Skipping]==> %d\n", len(rgbaData))
				continue
			}

			sectionImage.Pix = rgbaData

			draw.Draw(img, image.Rect(curX, curY, curX+int(section.Width), curY+int(section.Height)), sectionImage, image.Point{0, 0}, draw.Src)
			//data = append(data, rgbaData...)

			curX += int(section.Width)
		}
		curY += int(section.Height)
	}

	fmt.Printf("Loaded data with length %d\n", len(data))

	//img.Pix = data

	f, err = os.Create("full.png")
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
