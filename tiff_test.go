package tiff

import (
	"fmt"
	"log"
	"testing"
)

func TestLoad(t *testing.T) {
	filename := "C:\\Work\\PuffPiece\\kidney msi-if-imc_Scan1.qptiff"

	tiffFile, err := Open(filename)	
	if err != nil {
		log.Fatal(err)
	}
	defer tiffFile.Close()



	fmt.Println(tiffFile)
}