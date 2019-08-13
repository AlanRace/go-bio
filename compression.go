package tiff

import (
	"bytes"
	"compress/lzw"
	"io"
	"io/ioutil"

	"github.com/AlanRace/go-bio/jpeg"
)

var compressionNameMap = map[CompressionID]string{
	Uncompressed: "Uncompressed",
	CCIT1D:       "CCIT1D",
	CCITGroup3:   "CCITGroup3",
	CCITGroup4:   "CCITGroup4",
	LZW:          "LZW",
	OJPEG:        "OJPEG",
	JPEG:         "JPEG",
}

var compressionTypeMap = map[uint16]CompressionID{
	1: Uncompressed,
	2: CCIT1D,
	3: CCITGroup3,
	4: CCITGroup4,
	5: LZW,
	6: OJPEG,
	7: JPEG,
}

var compressionFuncMap = map[CompressionID]func(TagAccess) (CompressionMethod, error){}

func AddCompression(id CompressionID, name string, create func(TagAccess) (CompressionMethod, error)) {
	compressionNameMap[id] = name
	compressionTypeMap[uint16(id)] = id

	compressionFuncMap[id] = create
}

func init() {
	AddCompression(LZW, "LZW", func(dataAccess TagAccess) (CompressionMethod, error) {
		return &LZWCompression{}, nil
	})
	AddCompression(JPEG, "JPEG", func(dataAccess TagAccess) (CompressionMethod, error) {
		if dataAccess.GetTag(JPEGTables) != nil {
			tablesTag, ok := dataAccess.GetTag(JPEGTables).(*ByteTag)
			if !ok {
				return nil, &FormatError{msg: "JPEGTables not recorded as byte"}
			}

			r := bytes.NewReader(tablesTag.data)

			header, err := jpeg.DecodeHeader(r)
			if err != nil {
				return nil, err
			}

			return &JPEGCompression{header: header}, nil
		}

		return nil, &FormatError{msg: "No JPEGTables tag, unsupported form of JPEG compression"}
	})
}

// CompressionMethod is an interface for decompressing a io.Reader to a []byte.
type CompressionMethod interface {
	Decompress(io.Reader) ([]byte, error)
}

// LZWCompression performs LZW decompression of data.
type LZWCompression struct {
}

// Decompress decompresses an io.Reader using the LZW algorithm.
func (*LZWCompression) Decompress(r io.Reader) ([]byte, error) {
	readCloser := lzw.NewReader(r, lzw.MSB, 8)
	defer readCloser.Close()

	return ioutil.ReadAll(readCloser)
}

// JPEGCompression
type JPEGCompression struct {
	header *jpeg.JPEGHeader
}

// Decompress decompresses an io.Reader using the JPEG algorithm.
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
