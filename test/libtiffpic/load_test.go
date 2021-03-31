package test

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	tiff "github.com/AlanRace/go-bio"
	tiffimage "github.com/AlanRace/go-bio/image"
)

func TestLoad(t *testing.T) {

	err := filepath.Walk("/home/alan/Documents/Nicole/Andreas/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match("*.tif", filepath.Base(path)); err != nil {
			return err
		} else if matched {
			log.Printf("Processing %s\n", path)
			canContinue := true
			tiffFile, err := tiff.Open(path)
			if err != nil {
				log.Printf("[ERROR] %v", err)
				canContinue = false
			}
			defer tiffFile.Close()

			if canContinue {
				for index, ifd := range tiffFile.GetIFDList() {
					width, height := ifd.GetImageDimensions()
					if width > 10000 || height > 10000 {
						continue
					}

					outputImage := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

					secWidth, secHeight := ifd.GetSectionDimensions()
					secX, secY := ifd.GetSectionGrid()

					curX := 0
					curY := 0

					maxValue := float32(0)
					minValue := float32(1e20)

					for y := 0; y < int(secY); y++ {
						for x := 0; x < int(secX); x++ {
							section := ifd.GetSectionAt(int64(curX), int64(curY))

							defer func() {
								if r := recover(); r != nil {
									log.Printf("[PANIC] when processing IFD %d:  %v", index, r)
								}
							}()

							img, err := section.GetImage()
							if err != nil {
								log.Printf("[ERROR] when processing IFD %d:  %v", index, err)
								continue
							}

							if floatimg, ok := img.(*tiffimage.GrayFloat32); ok {
								if floatimg.MaxValue > maxValue {
									maxValue = floatimg.MaxValue
								}
							}

							curX += int(secWidth)
						}

						curX = 0
						curY += int(secHeight)
					}

					fmt.Printf("Found maxValue of %f and minValue of %f\n", maxValue, minValue)

					curX = 0
					curY = 0

					for y := 0; y < int(secY); y++ {
						for x := 0; x < int(secX); x++ {
							section := ifd.GetSectionAt(int64(curX), int64(curY))

							defer func() {
								if r := recover(); r != nil {
									log.Printf("[PANIC] when processing IFD %d:  %v", index, r)
								}
							}()

							img, err := section.GetImage()
							if err != nil {
								log.Printf("[ERROR] when processing IFD %d:  %v", index, err)
								continue
							}

							if floatimg, ok := img.(*tiffimage.GrayFloat32); ok {
								floatimg.MaxValue = maxValue

								draw.Draw(outputImage, image.Rect(curX, curY, curX+int(secWidth), curY+int(secHeight)), floatimg, image.Point{0, 0}, draw.Src)
							} else {
								draw.Draw(outputImage, image.Rect(curX, curY, curX+int(secWidth), curY+int(secHeight)), img, image.Point{0, 0}, draw.Src)
							}

							curX += int(secWidth)
						}

						curX = 0
						curY += int(secHeight)
					}

					f, err := os.Create(path[:len(path)-4] + "_IFD_" + strconv.Itoa(index) + ".png")
					if err != nil {
						panic(err)
					}
					defer f.Close()
					png.Encode(f, outputImage)

					log.Printf("[SUCCESS] when processing IFD %d", index)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
