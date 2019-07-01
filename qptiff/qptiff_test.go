package qptiff

import (
	"log"
	"testing"
	"fmt"
)

func TestLoad(t *testing.T) {
	filename := "C:\\Work\\PuffPiece\\kidney msi-if-imc_Scan1.qptiff"

	qptiffFile, err := Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer qptiffFile.Close()

	fmt.Println(qptiffFile)
}
