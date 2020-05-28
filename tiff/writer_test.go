package tiff

import (
	"fmt"
	"testing"
)

type TestImage struct {
}

func (image *TestImage) Width() uint32 {
	return 100
}

func (image *TestImage) Height() uint32 {
	return 100
}
func (image *TestImage) PixelAt(x, y uint32) []float64 {
	return []float64{float64(x + y)}
}

func TestCreate(t *testing.T) {
	writer, err := Create("test.tiff")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(writer)

	var image TestImage

	ifdWriter := writer.NewTiledIFD(&image)
	ifdWriter.SetResolution(0.7, 0.7, Centimeter)
	ifdWriter.SetTileSize(16, 16)
	ifdWriter.Write()
}
