package jpeg

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const (
	Marker uint8 = 0xff
	SOI    uint8 = 0xd8
	APP0   uint8 = 0xe0
	// ...
	DQT  uint8 = 0xdb
	SOF0 uint8 = 0xc0
	DHT  uint8 = 0xc4
	SOS  uint8 = 0xda
	EOI  uint8 = 0xd9

	RST0 uint8 = 0xd0
	RST7 uint8 = 0xd7
)

type QuantizationTable8Bit struct {
	QTNumber  uint8
	Precision uint8
	Values    []uint8
}

// LUT generation taken from https://golang.google.cn/src/image/jpeg/huffman.go

// maxCodeLength is the maximum (inclusive) number of bits in a Huffman code.
const maxCodeLength = 16

// maxNCodes is the maximum (inclusive) number of codes in a Huffman tree.
const maxNCodes = 256

// lutSize is the log-2 size of the Huffman decoder's look-up table.
const lutSize = 8

type HuffmanTable struct {
	NumberOfSymbols [16]uint8
	Symbols         []uint8

	// length is the number of codes in the tree.
	nCodes uint8
	// lut is the look-up table for the next lutSize bits in the bit-stream.
	// The high 8 bits of the uint16 are the encoded value. The low 8 bits
	// are 1 plus the code length, or 0 if the value is too large to fit in
	// lutSize bits.
	lut [1 << lutSize]uint16
	// minCodes[i] is the minimum code of length i, or -1 if there are no
	// codes of that length.
	minCodes [maxCodeLength]int32
	// maxCodes[i] is the maximum code of length i, or -1 if there are no
	// codes of that length.
	maxCodes [maxCodeLength]int32
	// valsIndices[i] is the index into vals of minCodes[i].
	valsIndices [maxCodeLength]int32
}

func (ht *HuffmanTable) generateLUT() {
	// Derive the look-up table.
	for i := range ht.lut {
		ht.lut[i] = 0
	}
	var x, code uint32
	for i := uint32(0); i < lutSize; i++ {
		code <<= 1
		for j := uint8(0); j < ht.NumberOfSymbols[i]; j++ {
			// The codeLength is 1+i, so shift code by 8-(1+i) to
			// calculate the high bits for every 8-bit sequence
			// whose codeLength's high bits matches code.
			// The high 8 bits of lutValue are the encoded value.
			// The low 8 bits are 1 plus the codeLength.
			base := uint8(code << (7 - i))
			lutValue := uint16(ht.Symbols[x])<<8 | uint16(2+i)
			for k := uint8(0); k < 1<<(7-i); k++ {
				ht.lut[base|k] = lutValue
			}
			code++
			x++
		}
	}

	// Derive minCodes, maxCodes, and valsIndices.
	var c, index int32
	for i, n := range ht.NumberOfSymbols {
		if n == 0 {
			ht.minCodes[i] = -1
			ht.maxCodes[i] = -1
			ht.valsIndices[i] = -1
		} else {
			ht.minCodes[i] = c
			ht.maxCodes[i] = c + int32(n) - 1
			ht.valsIndices[i] = index
			c += int32(n)
			index += int32(n)
		}
		c <<= 1
	}
}

type JPEGHeader struct {
	dqt QuantizationTable8Bit
	// First dimension is DC (0) or AC (1). Second dimension is  HT number
	huffmanTables [2][4]HuffmanTable
}

type FrameComponent struct {
	ComponentID              uint8
	VerticalSamplingFactor   uint8
	HorizontalSamplingFactor uint8
	QuantizationTableNumber  uint8
}

type StartOfFrame struct {
	Length        uint16
	Precision     uint8
	ImageHeight   uint16
	ImageWidth    uint16
	NumComponents uint8
}

type ScanComponent struct {
	ComponentID uint8
	ACTable     *HuffmanTable
	DCTable     *HuffmanTable
}

type JPEGBody struct {
	sof0            StartOfFrame
	frameComponents []FrameComponent
	scanComponents  []ScanComponent
}

func decodeHeader(r io.Reader) (*JPEGHeader, error) {
	var header JPEGHeader
	var buf []byte
	var err error
	var n int

	buf = make([]byte, 4028)

	n, err = r.Read(buf[:2])
	if err != nil {
		return nil, err
	}

	for n > 0 {
		// Process data
		if buf[0] == Marker {
			switch buf[1] {
			case SOI:
				fmt.Println("Found SOI")
			case EOI:
				fmt.Println("Found EOI")
			case DQT:
				fmt.Println("Found DQT")
				var table QuantizationTable8Bit
				type dqt struct {
					Length      uint16
					Information uint8
				}
				var qt dqt

				err = binary.Read(r, binary.BigEndian, &qt)
				if err != nil {
					return nil, err
				}

				table.QTNumber = qt.Information & 0x0F
				table.Precision = (qt.Information & 0xF0) >> 4
				table.Values = make([]uint8, 64)

				err = binary.Read(r, binary.BigEndian, &table.Values)
				if err != nil {
					return nil, err
				}

				header.dqt = table
			case DHT:
				fmt.Println("Found DHT")
				type dht struct {
					Length          uint16
					Information     uint8
					NumberOfSymbols [16]uint8
				}

				var ht dht

				err = binary.Read(r, binary.BigEndian, &ht)
				if err != nil {
					return nil, err
				}

				htNumber := ht.Information & 0x0F
				htType := (ht.Information & 16) >> 4

				copy(header.huffmanTables[htType][htNumber].NumberOfSymbols[:], ht.NumberOfSymbols[:])

				for _, num := range ht.NumberOfSymbols {
					header.huffmanTables[htType][htNumber].nCodes += num
				}

				header.huffmanTables[htType][htNumber].Symbols = make([]uint8, header.huffmanTables[htType][htNumber].nCodes)

				err = binary.Read(r, binary.BigEndian, &header.huffmanTables[htType][htNumber].Symbols)
				if err != nil {
					return nil, err
				}

				header.huffmanTables[htType][htNumber].generateLUT()
			default:
				fmt.Printf("Found marker: %s\n", hex.EncodeToString(buf[:2]))
			}

		}

		n, err = r.Read(buf[:2])
		if err != nil {
			return &header, err
		}
	}

	return &header, nil
}

func decodeBody(r io.Reader, header *JPEGHeader) error {
	var body JPEGBody
	var buf []byte
	var err error
	var n int

	buf = make([]byte, 4028)

	n, err = r.Read(buf[:2])
	if err != nil {
		return err
	}

	done := false

	for n > 0 {
		// Process data
		if buf[0] == Marker {
			switch buf[1] {
			case SOI:
				fmt.Println("Found SOI")
			case EOI:
				fmt.Println("Found EOI")
			case SOF0:
				fmt.Println("Found SOF0")
				err = binary.Read(r, binary.BigEndian, &body.sof0)
				if err != nil {
					return err
				}

				body.frameComponents = make([]FrameComponent, body.sof0.NumComponents)

				n, err = r.Read(buf[:(body.sof0.NumComponents * 3)])
				if err != nil {
					return err
				}

				for index := range body.frameComponents {
					offset := index * 3

					body.frameComponents[index].ComponentID = buf[offset]
					body.frameComponents[index].VerticalSamplingFactor = buf[offset+1] & 0x0F
					body.frameComponents[index].HorizontalSamplingFactor = (buf[offset+1] & 0xF0) >> 4
					body.frameComponents[index].QuantizationTableNumber = buf[offset+2]
				}
			case SOS:
				fmt.Println("Found SOS")
				n, err = r.Read(buf[:3])
				if err != nil {
					return err
				}

				body.scanComponents = make([]ScanComponent, buf[2])

				n, err = r.Read(buf[:(buf[2]*2)+3])
				if err != nil {
					return err
				}

				for index := range body.scanComponents {
					offset := index * 2

					body.scanComponents[index].ComponentID = buf[offset]
					acTableIndex := buf[offset+1] & 0x0F
					dcTableIndex := (buf[offset+1] & 0xF0) >> 4

					body.scanComponents[index].ACTable = &header.huffmanTables[1][acTableIndex]
					body.scanComponents[index].DCTable = &header.huffmanTables[0][dcTableIndex]
				}

				decodeSOS(header, &body, r)
			default:
				fmt.Printf("Found marker: %s\n", hex.EncodeToString(buf[:2]))
			}
		} else {
			if !done {
				fmt.Printf("Found unknown data: %s\n", hex.EncodeToString(buf[:2]))
				done = true
			}
		}

		n, err = r.Read(buf[:2])
		if err != nil {
			fmt.Println(body)

			return err
		}
	}

	fmt.Println(body)

	/*if _, err := r.Read(buf); err != nil {
		return err
	}*/

	//fmt.Println(hex.EncodeToString(buf[:2]))

	return nil
}

// bits holds the unprocessed bits that have been taken from the byte-stream.
// The n least significant bits of a form the unread bits, to be read in MSB to
// LSB order.
type bits struct {
	a uint32 // accumulator.
	m uint32 // mask. m==1<<(n-1) when n>0, with m==0 when n==0.
	n int32  // the number of unread bits in a.
}

// A FormatError reports that the input is not a valid JPEG.
type FormatError string

func (e FormatError) Error() string { return "invalid JPEG format: " + string(e) }

// An UnsupportedError reports that the input uses a valid but unimplemented JPEG feature.
type UnsupportedError string

func (e UnsupportedError) Error() string { return "unsupported JPEG feature: " + string(e) }

const (
	dcTable = 0
	acTable = 1
	maxTc   = 1
	maxTh   = 3
	maxTq   = 3

	maxComponents = 4
)

type decoder struct {
	header *JPEGHeader
	body   *JPEGBody
	r      io.Reader

	data [256][256][3]uint8

	bits bits
	// bytes is a byte buffer, similar to a bufio.Reader, except that it
	// has to be able to unread more than 1 byte, due to byte stuffing.
	// Byte stuffing is specified in section F.1.2.3.
	bytes struct {
		// buf[i:j] are the buffered bytes read from the underlying
		// io.Reader that haven't yet been passed further on.
		buf  [4096]byte
		i, j int
		// nUnreadable is the number of bytes to back up i after
		// overshooting. It can be 0, 1 or 2.
		nUnreadable int
	}

	ri    int // Restart Interval.
	nComp int

	// As per section 4.5, there are four modes of operation (selected by the
	// SOF? markers): sequential DCT, progressive DCT, lossless and
	// hierarchical, although this implementation does not support the latter
	// two non-DCT modes. Sequential DCT is further split into baseline and
	// extended, as per section 4.11.
	baseline    bool
	progressive bool

	jfif                bool
	adobeTransformValid bool
	adobeTransform      uint8
	eobRun              uint16 // End-of-Band run, specified in section G.1.2.2.

	tmp [2 * blockSize]byte
}

// refine decodes a successive approximation refinement block, as specified in
// section G.1.2.
func (d *decoder) refine(b *block, h *HuffmanTable, zigStart, zigEnd, delta int32) error {
	// Refining a DC component is trivial.
	if zigStart == 0 {
		if zigEnd != 0 {
			panic("unreachable")
		}
		bit, err := d.decodeBit()
		if err != nil {
			return err
		}
		if bit {
			b[0] |= delta
		}
		return nil
	}

	// Refining AC components is more complicated; see sections G.1.2.2 and G.1.2.3.
	zig := zigStart
	if d.eobRun == 0 {
	loop:
		for ; zig <= zigEnd; zig++ {
			z := int32(0)
			value, err := d.decodeHuffman(h)
			if err != nil {
				return err
			}
			val0 := value >> 4
			val1 := value & 0x0f

			switch val1 {
			case 0:
				if val0 != 0x0f {
					d.eobRun = uint16(1 << val0)
					if val0 != 0 {
						bits, err := d.decodeBits(int32(val0))
						if err != nil {
							return err
						}
						d.eobRun |= uint16(bits)
					}
					break loop
				}
			case 1:
				z = delta
				bit, err := d.decodeBit()
				if err != nil {
					return err
				}
				if !bit {
					z = -z
				}
			default:
				return FormatError("unexpected Huffman code")
			}

			zig, err = d.refineNonZeroes(b, zig, zigEnd, int32(val0), delta)
			if err != nil {
				return err
			}
			if zig > zigEnd {
				return FormatError("too many coefficients")
			}
			if z != 0 {
				b[unzig[zig]] = z
			}
		}
	}
	if d.eobRun > 0 {
		d.eobRun--
		if _, err := d.refineNonZeroes(b, zig, zigEnd, -1, delta); err != nil {
			return err
		}
	}
	return nil
}

func decodeSOS(header *JPEGHeader, body *JPEGBody, r io.Reader) error {
	var buf []byte
	var err error
	// TODO: We make this twice...
	buf = make([]byte, 4028)

	var d decoder
	d.header = header
	d.body = body
	d.r = r

	zigStart, zigEnd, ah, al := int32(0), int32(blockSize-1), uint32(0), uint32(0)

	// mxx and myy are the number of MCUs (Minimum Coded Units) in the image.
	h0, v0 := uint16(body.frameComponents[0].HorizontalSamplingFactor), uint16(body.frameComponents[0].VerticalSamplingFactor) // The h and v values from the Y components.
	mxx := int((body.sof0.ImageWidth + 8*h0 - 1) / (8 * h0))
	myy := int((body.sof0.ImageHeight + 8*v0 - 1) / (8 * v0))

	/*if d.img1 == nil && d.img3 == nil {
		d.makeImg(mxx, myy)
	}*/

	fmt.Printf("h0 = %d, v0 = %d, mxx = %d, myy = %d\n", h0, v0, mxx, myy)

	d.bits = bits{}
	mcu, expectedRST := 0, uint8(RST0)
	var (
		// b is the decoded coefficients, in natural (not zig-zag) order.
		b  block
		dc [maxComponents]int32
		// bx and by are the location of the current block, in units of 8x8
		// blocks: the third block in the first row has (bx, by) = (2, 0).
		bx, by     int
		blockCount int
	)

	nComp := len(body.scanComponents)

	for my := 0; my < myy; my++ {
		for mx := 0; mx < mxx; mx++ {
			for i := 0; i < nComp; i++ {
				compIndex := body.scanComponents[i].ComponentID
				hi := int(body.frameComponents[compIndex].HorizontalSamplingFactor)
				vi := int(body.frameComponents[compIndex].VerticalSamplingFactor)
				for j := 0; j < hi*vi; j++ {
					// The blocks are traversed one MCU at a time. For 4:2:0 chroma
					// subsampling, there are four Y 8x8 blocks in every 16x16 MCU.
					//
					// For a sequential 32x16 pixel image, the Y blocks visiting order is:
					//	0 1 4 5
					//	2 3 6 7
					//
					// For progressive images, the interleaved scans (those with nComp > 1)
					// are traversed as above, but non-interleaved scans are traversed left
					// to right, top to bottom:
					//	0 1 2 3
					//	4 5 6 7
					// Only DC scans (zigStart == 0) can be interleaved. AC scans must have
					// only one component.
					//
					// To further complicate matters, for non-interleaved scans, there is no
					// data for any blocks that are inside the image at the MCU level but
					// outside the image at the pixel level. For example, a 24x16 pixel 4:2:0
					// progressive image consists of two 16x16 MCUs. The interleaved scans
					// will process 8 Y blocks:
					//	0 1 4 5
					//	2 3 6 7
					// The non-interleaved scans will process only 6 Y blocks:
					//	0 1 2
					//	3 4 5
					if nComp != 1 {
						bx = hi*mx + j%hi
						by = vi*my + j/hi
					} else {
						q := mxx * hi
						bx = blockCount % q
						by = blockCount / q
						blockCount++
						if bx*8 >= int(body.sof0.ImageWidth) || by*8 >= int(body.sof0.ImageHeight) {
							continue
						}
					}

					b = block{}

					if ah != 0 {
						if err := d.refine(&b, body.scanComponents[i].ACTable, zigStart, zigEnd, 1<<al); err != nil {
							return err
						}
					} else {
						zig := zigStart
						if zig == 0 {
							zig++
							// Decode the DC coefficient, as specified in section F.2.2.1.
							value, err := d.decodeHuffman(body.scanComponents[i].DCTable)
							if err != nil {
								return err
							}
							if value > 16 {
								return UnsupportedError("excessive DC component")
							}
							dcDelta, err := d.receiveExtend(value)
							if err != nil {
								return err
							}
							dc[compIndex] += dcDelta
							b[0] = dc[compIndex] << al
						}

						if zig <= zigEnd && d.eobRun > 0 {
							d.eobRun--
						} else {
							// Decode the AC coefficients, as specified in section F.2.2.2.
							huff := body.scanComponents[i].ACTable
							for ; zig <= zigEnd; zig++ {
								value, err := d.decodeHuffman(huff)
								if err != nil {
									return err
								}
								val0 := value >> 4
								val1 := value & 0x0f
								if val1 != 0 {
									zig += int32(val0)
									if zig > zigEnd {
										break
									}
									ac, err := d.receiveExtend(val1)
									if err != nil {
										return err
									}
									b[unzig[zig]] = ac << al
								} else {
									if val0 != 0x0f {
										d.eobRun = uint16(1 << val0)
										if val0 != 0 {
											bits, err := d.decodeBits(int32(val0))
											if err != nil {
												return err
											}
											d.eobRun |= uint16(bits)
										}
										d.eobRun--
										break
									}
									zig += 0x0f
								}
							}
						}
					}

					if err := d.reconstructBlock(&b, bx, by, int(compIndex)); err != nil {
						return err
					}
				} // for j
			} // for i

			//fmt.Println(fmt.Sprintf("Processed MCU %d with RST %x", mcu, expectedRST))

			mcu++
			if d.ri > 0 && mcu%d.ri == 0 && mcu < int(mxx)*int(myy) {
				// A more sophisticated decoder could use RST[0-7] markers to resynchronize from corrupt input,
				// but this one assumes well-formed input, and hence the restart marker follows immediately.
				_, err = r.Read(buf[:2])
				if err != nil {
					return err
				}
				if buf[0] != 0xff || buf[1] != expectedRST {
					return FormatError("bad RST marker (expected " + fmt.Sprintf("%x", expectedRST) + " but got " + fmt.Sprintf("%x", buf[1]) + ")")
				}
				expectedRST++
				if expectedRST == RST7+1 {
					expectedRST = RST0
				}
				// Reset the Huffman decoder.
				d.bits = bits{}
				// Reset the DC components, as per section F.2.1.3.1.
				dc = [maxComponents]int32{}
				// Reset the progressive decoder state, as per section G.1.2.2.
				d.eobRun = 0
			}
		} // for mx
	} // for my

	fmt.Println(d.data)

	f, err := os.Create("dmp.bin")
	if err != nil {
		return err
	}
	defer f.Close()

	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			_, err := f.Write(d.data[y][x][:])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// unzig maps from the zig-zag ordering to the natural ordering. For example,
// unzig[3] is the column and row of the fourth element in zig-zag order. The
// value is 16, which means first column (16%8 == 0) and third row (16/8 == 2).
var unzig = [blockSize]int{
	0, 1, 8, 16, 9, 2, 3, 10,
	17, 24, 32, 25, 18, 11, 4, 5,
	12, 19, 26, 33, 40, 48, 41, 34,
	27, 20, 13, 6, 7, 14, 21, 28,
	35, 42, 49, 56, 57, 50, 43, 36,
	29, 22, 15, 23, 30, 37, 44, 51,
	58, 59, 52, 45, 38, 31, 39, 46,
	53, 60, 61, 54, 47, 55, 62, 63,
}

// readByteStuffedByte is like readByte but is for byte-stuffed Huffman data.
func (d *decoder) readByteStuffedByte() (x byte, err error) {
	// Take the fast path if d.bytes.buf contains at least two bytes.
	if d.bytes.i+2 <= d.bytes.j {
		x = d.bytes.buf[d.bytes.i]
		d.bytes.i++
		d.bytes.nUnreadable = 1
		if x != 0xff {
			return x, err
		}
		if d.bytes.buf[d.bytes.i] != 0x00 {
			return 0, errMissingFF00
		}
		d.bytes.i++
		d.bytes.nUnreadable = 2
		return 0xff, nil
	}

	d.bytes.nUnreadable = 0

	x, err = d.readByte()
	if err != nil {
		return 0, err
	}
	d.bytes.nUnreadable = 1
	if x != 0xff {
		return x, nil
	}

	x, err = d.readByte()
	if err != nil {
		return 0, err
	}
	d.bytes.nUnreadable = 2
	if x != 0x00 {
		return 0, errMissingFF00
	}
	return 0xff, nil
}

// errMissingFF00 means that readByteStuffedByte encountered an 0xff byte (a
// marker byte) that wasn't the expected byte-stuffed sequence 0xff, 0x00.
var errMissingFF00 = FormatError("missing 0xff00 sequence")

// unreadByteStuffedByte undoes the most recent readByteStuffedByte call,
// giving a byte of data back from d.bits to d.bytes. The Huffman look-up table
// requires at least 8 bits for look-up, which means that Huffman decoding can
// sometimes overshoot and read one or two too many bytes. Two-byte overshoot
// can happen when expecting to read a 0xff 0x00 byte-stuffed byte.
func (d *decoder) unreadByteStuffedByte() {
	d.bytes.i -= d.bytes.nUnreadable
	d.bytes.nUnreadable = 0
	if d.bits.n >= 8 {
		d.bits.a >>= 8
		d.bits.n -= 8
		d.bits.m >>= 8
	}
}

// refineNonZeroes refines non-zero entries of b in zig-zag order. If nz >= 0,
// the first nz zero entries are skipped over.
func (d *decoder) refineNonZeroes(b *block, zig, zigEnd, nz, delta int32) (int32, error) {
	for ; zig <= zigEnd; zig++ {
		u := unzig[zig]
		if b[u] == 0 {
			if nz == 0 {
				break
			}
			nz--
			continue
		}
		bit, err := d.decodeBit()
		if err != nil {
			return 0, err
		}
		if !bit {
			continue
		}
		if b[u] >= 0 {
			b[u] += delta
		} else {
			b[u] -= delta
		}
	}
	return zig, nil
}

// readByte returns the next byte, whether buffered or not buffered. It does
// not care about byte stuffing.
func (d *decoder) readByte() (x byte, err error) {
	for d.bytes.i == d.bytes.j {
		if err = d.fill(); err != nil {
			return 0, err
		}
	}
	x = d.bytes.buf[d.bytes.i]
	d.bytes.i++
	d.bytes.nUnreadable = 0
	return x, nil
}

// fill fills up the d.bytes.buf buffer from the underlying io.Reader. It
// should only be called when there are no unread bytes in d.bytes.
func (d *decoder) fill() error {
	if d.bytes.i != d.bytes.j {
		panic("jpeg: fill called when unread bytes exist")
	}
	// Move the last 2 bytes to the start of the buffer, in case we need
	// to call unreadByteStuffedByte.
	if d.bytes.j > 2 {
		d.bytes.buf[0] = d.bytes.buf[d.bytes.j-2]
		d.bytes.buf[1] = d.bytes.buf[d.bytes.j-1]
		d.bytes.i, d.bytes.j = 2, 2
	}
	// Fill in the rest of the buffer.
	n, err := d.r.Read(d.bytes.buf[d.bytes.j:])
	d.bytes.j += n
	if n > 0 {
		err = nil
	}
	return err
}

// reconstructBlock dequantizes, performs the inverse DCT and stores the block
// to the image.
func (d *decoder) reconstructBlock(b *block, bx, by, compIndex int) error {
	fmt.Printf("Reconstructing block... %d %d %d", bx, by, compIndex)
	fmt.Println(b)

	//qt := &d.header.dqt[d.body.frameComponents[compIndex].ComponentID]
	qt := d.header.dqt.Values
	for zig := 0; zig < blockSize; zig++ {
		b[unzig[zig]] *= int32(qt[zig])
	}
	idct(b)

	for y := 0; y < 8; y++ {
		y8 := y * 8
		for x := 0; x < 8; x++ {
			c := b[y8+x]

			if c < -128 {
				c = 0
			} else if c > 127 {
				c = 255
			} else {
				c += 128
			}

			d.data[by*8+y][bx*8+x][compIndex] = uint8(c)
		}
	}

	fmt.Println(b)
	/*dst, stride := []byte(nil), 0
	if d.nComp == 1 {
		dst, stride = d.img1.Pix[8*(by*d.img1.Stride+bx):], d.img1.Stride
	} else {
		switch compIndex {
		case 0:
			dst, stride = d.img3.Y[8*(by*d.img3.YStride+bx):], d.img3.YStride
		case 1:
			dst, stride = d.img3.Cb[8*(by*d.img3.CStride+bx):], d.img3.CStride
		case 2:
			dst, stride = d.img3.Cr[8*(by*d.img3.CStride+bx):], d.img3.CStride
		case 3:
			dst, stride = d.blackPix[8*(by*d.blackStride+bx):], d.blackStride
		default:
			return UnsupportedError("too many components")
		}
	}*/

	// Level shift by +128, clip to [0, 255], and write to dst.
	/*for y := 0; y < 8; y++ {
		y8 := y * 8
		yStride := y * stride
		for x := 0; x < 8; x++ {
			c := b[y8+x]
			if c < -128 {
				c = 0
			} else if c > 127 {
				c = 255
			} else {
				c += 128
			}
			dst[yStride+x] = uint8(c)
		}
	}*/
	return nil
}
