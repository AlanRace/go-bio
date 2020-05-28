package main

import (
	"fmt"
	"log"
	"os"

	tiff "github.com/AlanRace/go-bio"

	// Make sure that we include SVS package so we correctly handle SVS files
	"github.com/AlanRace/go-bio/qptiff"
	_ "github.com/AlanRace/go-bio/svs"
)

func main() {
	if len(os.Args) > 1 {
		file, err := qptiff.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		fmt.Println(os.Args[1])
		fmt.Printf("Number of images: %d\n", len(file.IFDList))
		fmt.Println()

		for index, ifd := range file.IFDList {
			imageWidth, imageHeight := ifd.GetImageDimensions()

			bitsPerSample, err := ifd.GetBitsPerSample()
			if err != nil {
				fmt.Println(err)
				continue
			}

			samplesPerPixel, err := ifd.GetSamplesPerPixel()
			if err != nil {
				fmt.Println(err)
				continue
			}

			photometricInterpretation, err := ifd.GetPhotometricInterpretation()
			if err != nil {
				fmt.Println(err)
				continue
			}

			compression, err := ifd.GetCompression()
			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Printf("- Image %d\n", index)
			fmt.Printf("Image size: %d x %d\n", imageWidth, imageHeight)
			fmt.Printf("BitsPerSample: %d\n", bitsPerSample)
			fmt.Printf("SamplesPerPixel: %d\n", samplesPerPixel)
			fmt.Printf("PhotometricInterpretation: %s\n", photometricInterpretation.String())
			fmt.Printf("%s\n", ifd.GetTag(tiff.PlanarConfiguration))
			fmt.Printf("Compression: %s\n", compression.String())

			resTag := ifd.GetTag(tiff.XResolution)
			if resTag != nil {
				x, y, _, err := ifd.GetResolution()
				if err != nil {
					fmt.Println(err)
					continue
				}

				resolutionUnit, err := ifd.GetResolutionUnit()
				if err != nil {
					fmt.Println(err)
					continue
				}

				fmt.Printf("Resolution (%d): %f x %f\n", resolutionUnit, x, y)
			} else {
				fmt.Println("Resolution not defined.")
			}

			fmt.Printf("%s\n", ifd.GetTag(tiff.ImageDescription))
			fmt.Println()

			for tagID, tag := range ifd.Tags {
				if tag.NumItems() > 10 {
					fmt.Printf("%s (%d): [Array not printed]\n", tagID.String(), tagID)
				} else {
					fmt.Printf("%v\n", tag)
				}
			}

			/*if ifd.IsTiled() {
				dataAccess, ok := ifd.GetDataAccess().(*tiff.TileDataAccess)
				if !ok {
					fmt.Println("ERROR: Should be tiled, but isn't")
				}

				tileWidth, tileHeight := dataAccess.GetTileDimensions()
				fmt.Printf("Tiles: %d x %d\n", tileWidth, tileHeight)
			}*/

			fmt.Println()
		}
	}
}
