package image

import (
	"fmt"
	"image"
	"image/color"
	"math/bits"

	tiffcolor "github.com/AlanRace/go-bio/image/color"
)

// Implement Image interface for new structures describing different tiff options

// mul3NonNeg returns (x * y * z), unless at least one argument is negative or
// if the computation overflows the int type, in which case it returns -1.
func mul3NonNeg(x int, y int, z int) int {
	if (x < 0) || (y < 0) || (z < 0) {
		return -1
	}
	hi, lo := bits.Mul64(uint64(x), uint64(y))
	if hi != 0 {
		return -1
	}
	hi, lo = bits.Mul64(lo, uint64(z))
	if hi != 0 {
		return -1
	}
	a := int(lo)
	if (a < 0) || (uint64(a) != lo) {
		return -1
	}
	return a
}

// pixelBufferLength returns the length of the []uint8 typed Pix slice field
// for the NewXxx functions. Conceptually, this is just (bpp * width * height),
// but this function panics if at least one of those is negative or if the
// computation would overflow the int type.

//
// This panics instead of returning an error because of backwards
// compatibility. The NewXxx functions do not return an error.
func pixelBufferLength(bytesPerPixel int, r image.Rectangle, imageTypeName string) int {
	totalLength := mul3NonNeg(bytesPerPixel, r.Dx(), r.Dy())

	if totalLength < 0 {
		panic("image: New" + imageTypeName + " Rectangle has huge or negative dimensions")
	}

	return totalLength
}

type RGB struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func (p *RGB) ColorModel() color.Model { return tiffcolor.RGBModel }

func (p *RGB) Bounds() image.Rectangle { return p.Rect }

func (p *RGB) At(x, y int) color.Color {
	return p.RGBAt(x, y)
}

func (p *RGB) RGBAt(x, y int) tiffcolor.RGB {
	if !(image.Point{x, y}.In(p.Rect)) {
		return tiffcolor.RGB{}
	}

	i := p.PixOffset(x, y)

	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857f
	return tiffcolor.RGB{R: s[0], G: s[1], B: s[2]}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *RGB) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*3
}

func (p *RGB) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}

	i := p.PixOffset(x, y)
	c1 := tiffcolor.RGBModel.Convert(c).(tiffcolor.RGB)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857

	s[0] = c1.R
	s[1] = c1.G
	s[2] = c1.B
}

func (p *RGB) SetRGB(x, y int, c tiffcolor.RGB) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}

	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857

	s[0] = c.R
	s[1] = c.G
	s[2] = c.B
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *RGB) SubImage(r image.Rectangle) image.Image {

	r = r.Intersect(p.Rect)

	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &RGB{}
	}

	i := p.PixOffset(r.Min.X, r.Min.Y)

	return &RGB{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}

// NewRGB returns a new RGB image with the given bounds.
func NewRGB(r image.Rectangle) *RGB {
	return &RGB{
		Pix:    make([]uint8, pixelBufferLength(3, r, "RGB")),
		Stride: 3 * r.Dx(),
		Rect:   r,
	}
}

type RGB10 struct {
	Pix    []byte
	Stride int
	Rect   image.Rectangle
}

func (p *RGB10) ColorModel() color.Model { return tiffcolor.RGB16Model }

func (p *RGB10) Bounds() image.Rectangle { return p.Rect }

func (p *RGB10) At(x, y int) color.Color {
	return p.RGBAt(x, y)
}

var bitsRemaining = map[uint16]uint16{
	10: 0b1111111111,
	9:  0b1111111110,
	8:  0b1111111100,
	7:  0b1111111000,
	6:  0b1111110000,
	5:  0b1111100000,
	4:  0b1111000000,
	3:  0b1110000000,
	2:  0b1100000000,
	1:  0b1000000000,
}

func (p *RGB10) RGBAt(x, y int) tiffcolor.RGB16 {
	if !(image.Point{x, y}.In(p.Rect)) {
		return tiffcolor.RGB16{}
	}

	startByte, offset := p.PixOffset(x, y)
	s := p.Pix //[startByte : startByte+6 : startByte+6] // Small cap improves performance, see https://golang.org/issue/27857f

	remainingBits := uint16(10)
	var r, g, b uint16

	fmt.Printf("Processing %d, %d\n", x, y)

	for remainingBits > 0 && remainingBits <= 10 {
		for offset >= 8 {
			offset -= 8
			startByte++
		}

		temp := uint16(s[startByte]) >> offset
		r = r | (temp & bitsRemaining[remainingBits])

		processedBits := 8 - offset
		fmt.Printf("[R = %d] -> %d, %d, %d, %d\n", r, startByte, temp, remainingBits, offset)
		offset += processedBits

		if processedBits > remainingBits {
			break
		}
		remainingBits -= processedBits
	}

	remainingBits = 10

	for remainingBits > 0 && remainingBits <= 10 {
		for offset >= 8 {
			offset -= 8
			startByte++
		}

		temp := uint16(s[startByte]) >> offset
		g = g | (temp & bitsRemaining[remainingBits])

		processedBits := 8 - offset
		fmt.Printf("[G = %d] -> %d, %d, %d, %d, %d\n", g, startByte, temp, remainingBits, processedBits, offset)
		offset += processedBits

		if processedBits > remainingBits {
			break
		}
		remainingBits -= processedBits
	}

	remainingBits = 10

	for remainingBits > 0 && remainingBits <= 10 {
		for offset >= 8 {
			offset -= 8
			startByte++
		}

		temp := uint16(s[startByte]) >> offset
		b = b | (temp & bitsRemaining[remainingBits])

		processedBits := 8 - offset
		offset += processedBits

		fmt.Printf("[B = %d] -> %d, %d, %d, %d\n", b, startByte, temp, remainingBits, offset)
		if processedBits > remainingBits {
			break
		}
		remainingBits -= processedBits
	}

	fmt.Printf("(%d, %d) -> %d %d", x, y, startByte, offset)

	return tiffcolor.RGB16{R: r, G: g, B: b}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *RGB10) PixOffset(x, y int) (int, uint16) {
	index := (y-p.Rect.Min.Y)*(p.Rect.Max.X-p.Rect.Min.X) + (x - p.Rect.Min.X)
	numBits := index * 30
	remainder := numBits - ((numBits / 8) * 8)
	fmt.Println(index)
	fmt.Println(numBits)
	fmt.Println(numBits / 8)
	fmt.Println(remainder)
	return numBits / 8, uint16(remainder)
}
