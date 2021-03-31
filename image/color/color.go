package color

import (
	"image/color"
)

// Add additional colours not included in Go image/color

// Gray32 represents a 32-bit grayscale color.
type Gray32 struct {
	Y uint32
}

func (c Gray32) RGBA() (r, g, b, a uint32) {
	return c.Y, c.Y, c.Y, 0xffff
}

// RGB represents represents 3 byte color, having 8 bits for each of red, green and blue.
type RGB struct {
	R, G, B uint8
}

func (c RGB) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	r |= r << 8

	g = uint32(c.G)
	g |= g << 8

	b = uint32(c.B)
	b |= b << 8

	a = 0xffff

	return
}

// RGB16 represents represents 3 color, each having 16 bits for each of red, green and blue.
type RGB16 struct {
	R, G, B uint16
}

func (c RGB16) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	g = uint32(c.G)
	b = uint32(c.B)

	a = 0xffff

	return
}

var (
	Gray32Model color.Model = color.ModelFunc(gray32Model)
	RGBModel    color.Model = color.ModelFunc(rgbModel)
	RGB16Model  color.Model = color.ModelFunc(rgb16Model)
)

func gray32Model(c color.Color) color.Color {
	if _, ok := c.(Gray32); ok {
		return c
	}

	r, g, b, _ := c.RGBA()

	// These coefficients (the fractions 0.299, 0.587 and 0.114) are the same
	// as those given by the JFIF specification and used by func RGBToYCbCr in
	// ycbcr.go.
	//

	// Note that 19595 + 38470 + 7471 equals 65536.
	y := uint32(float64(r)*0.299 + float64(g)*0.587 + float64(b)*0.114)

	return Gray32{y}

}

func rgbModel(c color.Color) color.Color {
	if _, ok := c.(RGB); ok {
		return c
	}

	r, g, b, _ := c.RGBA()

	return RGB{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}

func rgb16Model(c color.Color) color.Color {
	if _, ok := c.(RGB16); ok {
		return c
	}

	r, g, b, _ := c.RGBA()

	return RGB16{uint16(r >> 16), uint16(g >> 8), uint16(b >> 16)}
}
