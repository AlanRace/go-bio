package svs

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"

	tiff "github.com/AlanRace/go-bio"
	"github.com/AlanRace/go-bio/jpeg2000"
)

const (
	YCbCrJPEG tiff.CompressionID = 33003
	RGBJPEG   tiff.CompressionID = 33005

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
	tiff.AddCompression(RGBJPEG, "JPEG (Aperio RGB)", func(dataAccess tiff.TagAccess) (tiff.CompressionMethod, error) {
		return &RGBJPEGCompression{}, nil
	})
}

func Open(path string) (*File, error) {
	var svsFile File
	tiffFile, err := tiff.Open(path)
	if err != nil {
		return nil, err
	}

	svsFile.File = *tiffFile

	return &svsFile, nil
}
