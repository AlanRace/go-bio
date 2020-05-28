package main

// To run, first install github.com/google/tiff with:
// go get github.com/google/tiff
// Then run like `go run tiff-dump.go example.tif`

import (
	"fmt"
	"os"

	"github.com/google/tiff"
)

func main() {

	tiffFile := os.Args[1]
	r, err := os.Open(tiffFile)
	if err != nil {
		panic(err)
	}

	tiff.SetTiffFieldPrintFullFieldValue(true)

	t, err := tiff.Parse(r, nil, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Version: %d\n", t.Version())
	fmt.Printf("Byte Order: %s\n\n", EndianString(t.Order()))

	for i, ifd := range t.IFDs() {
		fmt.Printf("IFD %d:\n", i)
		for _, f := range ifd.Fields() {
			fmt.Printf("%s\n", f)
		}
	}
}

func EndianString(magicEndian string) string {
	if magicEndian == tiff.MagicBigEndian {
		return "big endian"
	} else {
		return "little endian"
	}
}
