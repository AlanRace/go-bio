package main

import (
	"fmt"
	"log"
	"os"

	tiff "github.com/AlanRace/go-bio"
)

func main() {
	if len(os.Args) > 1 {
		file, err := tiff.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		fmt.Println(os.Args[1])
		fmt.Printf("Number of images: %d\n", len(file.IFDList))
		fmt.Println()

		for index, ifd := range file.IFDList {
			imageWidth, imageHeight := ifd.GetImageDimensions()

			fmt.Printf("- Image %d\n", index)
			fmt.Printf("Image size: %d x %d\n", imageWidth, imageHeight)
			fmt.Printf("BitsPerSample: %d\n", ifd.GetBitsPerSample())
			fmt.Printf("SamplesPerPixel: %d\n", ifd.GetSamplesPerPixel())
			fmt.Printf("PhotometricInterpretation: %s\n", ifd.GetPhotometricInterpretation().String())
			fmt.Printf("%s\n", ifd.GetTag(tiff.PlanarConfiguration))
			fmt.Printf("Compression: %s\n", ifd.GetCompression().String())

			resTag := ifd.GetTag(tiff.XResolution)
			if resTag != nil {
				x, y, _ := ifd.GetResolution()

				fmt.Printf("Resolution (%d): %f x %f\n", ifd.GetResolutionUnit(), x, y)
			}

			/*if ifd.IsTiled() {
				dataAccess, ok := ifd.GetDataAccess().(*tiff.TileDataAccess)
				if !ok {
					fmt.Println("ERROR: Should be tiled, but isn't")
				}

				tileWidth, tileHeight := dataAccess.GetTileDimensions()
				fmt.Printf("Tiles: %d x %d\n", tileWidth, tileHeight)
			}*/

			fmt.Printf("%s\n", ifd.GetTag(tiff.ImageDescription))
			fmt.Println()
		}
	}
}
