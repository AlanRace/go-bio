package gobio

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"

	"golang.org/x/image/tiff/lzw"

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
	PackBits:     "PackBits",
}

var compressionTypeMap = map[uint16]CompressionID{
	0:     Uncompressed,
	1:     Uncompressed,
	2:     CCIT1D,
	3:     CCITGroup3,
	4:     CCITGroup4,
	5:     LZW,
	6:     OJPEG,
	7:     JPEG,
	32773: PackBits,
}

var compressionFuncMap = map[CompressionID]func(TagAccess) (CompressionMethod, error){}

func AddCompression(id CompressionID, name string, create func(TagAccess) (CompressionMethod, error)) {
	compressionNameMap[id] = name
	compressionTypeMap[uint16(id)] = id

	compressionFuncMap[id] = create
}

func init() {

	AddCompression(Uncompressed, "Uncompressed", func(dataAccess TagAccess) (CompressionMethod, error) {
		return &NoCompression{}, nil
	})
	AddCompression(LZW, "LZW", func(dataAccess TagAccess) (CompressionMethod, error) {
		return &LZWCompression{}, nil
	})
	AddCompression(JPEG, "JPEG", func(dataAccess TagAccess) (CompressionMethod, error) {
		if dataAccess.GetTag(JPEGTables) != nil {
			tablesTag, ok := dataAccess.GetByteTag(JPEGTables) //.(*ByteTag)
			if !ok {
				return nil, &FormatError{msg: "JPEGTables not recorded as byte"}
			}

			r := bytes.NewReader(tablesTag.Data)

			header, err := jpeg.DecodeHeader(r)
			if err != nil {
				return nil, err
			}

			return &JPEGCompression{header: header}, nil
		}

		return nil, &FormatError{msg: "No JPEGTables tag, unsupported form of JPEG compression"}
	})
	AddCompression(PackBits, "PackBits", func(dataAccess TagAccess) (CompressionMethod, error) {
		return &PackBitsCompression{}, nil
	})
}

// CompressionMethod is an interface for decompressing a io.Reader to a []byte.
type CompressionMethod interface {
	Decompress(io.Reader) ([]byte, error)
}

// NoCompression performs no decompression of data.
type NoCompression struct {
}

// Decompress extracts bytes from io.Reader.
func (*NoCompression) Decompress(r io.Reader) ([]byte, error) {
	return ioutil.ReadAll(r)
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

type PackBitsCompression struct {
}

func (*PackBitsCompression) Decompress(r io.Reader) ([]byte, error) {
	return unpackBits(r)
}

type byteReader interface {
	io.Reader
	io.ByteReader
}

// unpackBits decodes the PackBits-compressed data in src and returns the
// uncompressed data.
//
// The PackBits compression format is described in section 9 (p. 42)
// of the TIFF spec.
func unpackBits(r io.Reader) ([]byte, error) {
	var n int
	buf := make([]byte, 128)
	dst := make([]byte, 0, 1024)
	br, ok := r.(byteReader)
	if !ok {
		br = bufio.NewReader(r)
	}

	for {
		b, err := br.ReadByte()
		if err != nil {
			if err == io.EOF {
				return dst, nil
			}
			return nil, err
		}
		code := int(int8(b))
		switch {
		case code >= 0:
			n, err = io.ReadFull(br, buf[:code+1])
			if err != nil {
				return nil, err
			}
			dst = append(dst, buf[:n]...)
		case code == -128:
			// No-op.
		default:
			if b, err = br.ReadByte(); err != nil {
				return nil, err
			}
			for j := 0; j < 1-code; j++ {
				buf[j] = b
			}
			dst = append(dst, buf[:1-code]...)
		}
	}
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
