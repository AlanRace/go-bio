package tiff

import (
	"compress/lzw"
	"io"
	"io/ioutil"

	"github.com/AlanRace/go-bio/jpeg"
)

type CompressionMethod interface {
	Decompress(io.Reader) ([]byte, error)
}

type LZWCompression struct {
}

func (*LZWCompression) Decompress(r io.Reader) ([]byte, error) {
	readCloser := lzw.NewReader(r, lzw.MSB, 8)
	defer readCloser.Close()

	return ioutil.ReadAll(readCloser)
}

type JPEGCompression struct {
	header *jpeg.JPEGHeader
}

func (c *JPEGCompression) Decompress(r io.Reader) ([]byte, error) {
	data, err := c.header.DecodeBody(r)
	if err != nil {
		return nil, err
	}

	return data.Data, nil
}

// Decompress decompresses the data supplied in the io.Reader using the compression method dictated by CompressionID.
/*func (compressionID CompressionID) Decompress(r io.Reader, ifd *ImageFileDirectory) ([]byte, error) {
	var uncompressedData []byte
	var err error

	switch compressionID {
	case LZW:
		readCloser := lzw.NewReader(r, lzw.MSB, 8)
		uncompressedData, err = ioutil.ReadAll(readCloser)
		readCloser.Close()
	case JPEG:
		img, err := jpeg.Decode(r)
		if err != nil {
			panic(err)
		}
		fmt.Println(img.At(0, 0).RGBA())
		f, err := os.Create("TESTDECODE.png")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		png.Encode(f, img)
		//uncompressedData = img.Pix
	default:
		return nil, &FormatError{msg: "Unsupported compression scheme: " + compressionID.String()}
	}

	return uncompressedData, err
}
*/
