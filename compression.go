package gobio

import (
	"bufio"
	"image"
	"io"
	"io/ioutil"

	"golang.org/x/image/tiff/lzw"
	//"github.com/AlanRace/go-bio/libjpeg"
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
	AddCompression(UndefinedCompression, "Uncompressed", func(dataAccess TagAccess) (CompressionMethod, error) {
		return &NoCompression{}, nil
	})
	AddCompression(Uncompressed, "Uncompressed", func(dataAccess TagAccess) (CompressionMethod, error) {
		return &NoCompression{}, nil
	})
	AddCompression(LZW, "LZW", func(dataAccess TagAccess) (CompressionMethod, error) {
		return &LZWCompression{}, nil
	})
	// AddCompression(JPEG, "JPEG", func(dataAccess TagAccess) (CompressionMethod, error) {
	// 	if dataAccess.GetTag(JPEGTables) != nil {
	// 		tablesTag, ok := dataAccess.GetByteTag(JPEGTables) //.(*ByteTag)
	// 		if !ok {
	// 			return nil, &FormatError{msg: "JPEGTables not recorded as byte"}
	// 		}

	// 		r := bytes.NewReader(tablesTag.Data)

	// 		ljpeg, err := libjpeg.NewJPEGFromHeader(r)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("Failed parsing JPEG header (libjpeg) %w", err)
	// 		}

	// 		return &JPEGCompression{ljpeg: ljpeg}, nil
	// 	}

	// 	return &JPEGCompression{ljpeg: libjpeg.NewJPEG()}, nil
	// })
	AddCompression(PackBits, "PackBits", func(dataAccess TagAccess) (CompressionMethod, error) {
		return &PackBitsCompression{}, nil
	})
}

// CompressionMethod is an interface for decompressing a io.Reader.
type CompressionMethod interface {
}

// BinaryDecompressor just returns binary data and doesn't know anything about an image. e.g. LZW
type BinaryDecompressor interface {
	CompressionMethod
	Decompress(io.Reader) ([]byte, error)
}

// ImageDecompressor returns an image as the information required to reconstruct this are included in the compressed data e.g. JPEG
type ImageDecompressor interface {
	CompressionMethod
	Decompress(io.Reader) (image.Image, error)
	SetPhotometricInterpretation(interpretation PhotometricInterpretationID)
}

// NoCompression performs no decompression of data.
type NoCompression struct {
	CompressionMethod
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

// // JPEGCompression
// type JPEGCompression struct {
// 	//header *jpeg.JPEGHeader

// 	ljpeg *libjpeg.JPEG
// }

// func NewJPEGCompression(ljpeg *libjpeg.JPEG) *JPEGCompression {
// 	return &JPEGCompression{ljpeg: ljpeg}
// }

// // Decompress decompresses an io.Reader using the JPEG algorithm.
// func (compression *JPEGCompression) Decompress(r io.Reader) (image.Image, error) {
// 	return compression.ljpeg.DecodeBody(r)
// }

// func (compression *JPEGCompression) SetPhotometricInterpretation(interpretation PhotometricInterpretationID) {
// 	switch interpretation {
// 	case RGB:
// 		compression.ljpeg.RGBColourSpace()
// 	case YCbCr:
// 		compression.ljpeg.YCbCrColourSpace()
// 	default:
// 		panic("Unsupported photometric interpretation !" + interpretation.String())
// 	}
// }

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
