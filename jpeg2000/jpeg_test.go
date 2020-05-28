package jpeg2000

import (
	"os"
	"testing"
)

func TestJpeg2000(t *testing.T) {
	path := "tile.j2k"

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	Decode(file)
}
