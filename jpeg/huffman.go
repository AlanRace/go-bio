package jpeg

import "io"

// errShortHuffmanData means that an unexpected EOF occurred while decoding
// Huffman data.
var errShortHuffmanData = FormatError("short Huffman data")

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

// decodeHuffman returns the next Huffman-coded value from the bit-stream,
// decoded according to h.
func (d *decoder) decodeHuffman(h *HuffmanTable) (uint8, error) {
	if h.nCodes == 0 {
		return 0, FormatError("uninitialized Huffman table")
	}

	if d.bits.n < 8 {
		if err := d.ensureNBits(8); err != nil {
			if err != errMissingFF00 && err != errShortHuffmanData {
				return 0, err
			}
			// There are no more bytes of data in this segment, but we may still
			// be able to read the next symbol out of the previously read bits.
			// First, undo the readByte that the ensureNBits call made.
			if d.bytes.nUnreadable != 0 {
				d.unreadByteStuffedByte()
			}
			goto slowPath
		}
	}
	if v := h.lut[(d.bits.a>>uint32(d.bits.n-lutSize))&0xff]; v != 0 {
		n := (v & 0xff) - 1
		d.bits.n -= int32(n)
		d.bits.m >>= n
		return uint8(v >> 8), nil
	}

slowPath:
	for i, code := 0, int32(0); i < maxCodeLength; i++ {
		if d.bits.n == 0 {
			if err := d.ensureNBits(1); err != nil {
				return 0, err
			}
		}
		if d.bits.a&d.bits.m != 0 {
			code |= 1
		}
		d.bits.n--
		d.bits.m >>= 1
		if code <= h.maxCodes[i] {
			return h.Symbols[h.valsIndices[i]+code-h.minCodes[i]], nil
		}
		code <<= 1
	}
	return 0, FormatError("bad Huffman code")
}

func (d *decoder) decodeBit() (bool, error) {
	if d.bits.n == 0 {
		if err := d.ensureNBits(1); err != nil {
			return false, err
		}
	}
	ret := d.bits.a&d.bits.m != 0
	d.bits.n--
	d.bits.m >>= 1
	return ret, nil
}

func (d *decoder) decodeBits(n int32) (uint32, error) {
	if d.bits.n < n {
		if err := d.ensureNBits(n); err != nil {
			return 0, err
		}
	}
	ret := d.bits.a >> uint32(d.bits.n-n)
	ret &= (1 << uint32(n)) - 1
	d.bits.n -= n
	d.bits.m >>= uint32(n)
	return ret, nil
}

// ensureNBits reads bytes from the byte buffer to ensure that d.bits.n is at
// least n. For best performance (avoiding function calls inside hot loops),
// the caller is the one responsible for first checking that d.bits.n < n.
func (d *decoder) ensureNBits(n int32) error {
	for {
		c, err := d.readByteStuffedByte()
		if err != nil {
			if err == io.EOF {
				return errShortHuffmanData
			}
			return err
		}
		d.bits.a = d.bits.a<<8 | uint32(c)
		d.bits.n += 8
		if d.bits.m == 0 {
			d.bits.m = 1 << 7
		} else {
			d.bits.m <<= 8
		}
		if d.bits.n >= n {
			break
		}
	}
	return nil
}

// receiveExtend is the composition of RECEIVE and EXTEND, specified in section
// F.2.2.1.
func (d *decoder) receiveExtend(t uint8) (int32, error) {
	if d.bits.n < int32(t) {
		if err := d.ensureNBits(int32(t)); err != nil {
			return 0, err
		}
	}
	d.bits.n -= int32(t)
	d.bits.m >>= t
	s := int32(1) << t
	x := int32(d.bits.a>>uint8(d.bits.n)) & (s - 1)
	if x < s>>1 {
		x += ((-1) << t) + 1
	}
	return x, nil
}
