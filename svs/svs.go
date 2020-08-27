package svs

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	tiff "github.com/AlanRace/go-bio"
	"github.com/AlanRace/go-bio/jpeg2000"
)

const (
	//YCbCrJPEG tiff.CompressionID = 33003
	//RGBJPEG   tiff.CompressionID = 33005

	ImageDepth tiff.TagID = 32997
)

// LZWCompression performs LZW decompression of data.
type RGBJPEGCompression struct {
}

// Decompress decompresses an io.Reader using the LZW algorithm.
func (*RGBJPEGCompression) Decompress(r io.Reader) ([]byte, error) {
	// todo:output

	toOutput, _ := ioutil.ReadAll(r)

	// open output file
	fo, err := os.Create("tile.j2k")
	if err != nil {
		panic(err)
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()
	// make a write buffer
	w := bufio.NewWriter(fo)
	// write a chunk
	if _, err := w.Write(toOutput); err != nil {
		panic(err)
	}
	w.Flush()

	jpeg2000.Decode(r)

	return ioutil.ReadAll(r)
}

type File struct {
	tiff.File
}

func init() {
	tiff.AddTag(ImageDepth, "ImageDepth")
	//tiff.AddCompression(RGBJPEG, "JPEG (Aperio RGB)", func(dataAccess tiff.TagAccess) (tiff.CompressionMethod, error) {
	//	return &RGBJPEGCompression{}, nil
	//})
}

func Open(path string) (*File, error) {
	var svsFile File
	tiffFile, err := tiff.Open(path)
	if err != nil {
		return nil, err
	}

	svsFile.File = *tiffFile

	mainIFD := svsFile.IFDList[0]

	details := strings.Split(mainIFD.GetTag(tiff.ImageDescription).String(), "|")

	primaryWidth, primaryHeight := mainIFD.GetImageDimensions()

	// First entry (details[0] contains: "115920x45243 [0,100 113331x45143] (256x256) JPEG/RGB Q=30")

	for _, detail := range details[1:] {
		splitDetail := strings.Split(detail, "=")
		tag := strings.TrimSpace(splitDetail[0])
		value := strings.TrimSpace(splitDetail[1])

		switch tag {
		case "MPP":
			mpp, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, err
			}

			fmt.Println(mpp)

			mainIFD.PutTag(tiff.NewShortTag(tiff.ResolutionUnit, []uint16{uint16(tiff.Centimeter)}))
			mainIFD.PutTag(tiff.NewRationalTag(tiff.XResolution, []tiff.RationalNumber{*tiff.NewRationalNumber(10000.0 / mpp)}))
			mainIFD.PutTag(tiff.NewRationalTag(tiff.YResolution, []tiff.RationalNumber{*tiff.NewRationalNumber(10000.0 / mpp)}))

			for i := 1; i < svsFile.NumReducedImages(); i++ {
				ifd := svsFile.GetReducedImage(i)

				ifd.PutTag(tiff.NewShortTag(tiff.ResolutionUnit, []uint16{uint16(tiff.Centimeter)}))

				width, height := ifd.GetImageDimensions()

				ifd.PutTag(tiff.NewRationalTag(tiff.XResolution, []tiff.RationalNumber{*tiff.NewRationalNumber(10000.0 / (mpp * float64(primaryWidth) / float64(width)))}))
				ifd.PutTag(tiff.NewRationalTag(tiff.YResolution, []tiff.RationalNumber{*tiff.NewRationalNumber(10000.0 / (mpp * float64(primaryHeight) / float64(height)))}))
			}
		}
	}

	return &svsFile, nil
}

func (file File) NumReducedImages() int {
	return 5
}

func (file File) GetReducedImage(index int) *tiff.ImageFileDirectory {
	if index == 0 {
		return file.IFDList[0]
	}

	// The second image in the list is the lowest resolution image
	if index == 4 {
		return file.IFDList[1]
	}

	// Otherwise, they are in descending order
	return file.IFDList[index+1]
}
