package svs

import (
	"encoding/hex"
	"fmt"
	"log"
	"testing"
)

func TestLoad(t *testing.T) {
	filename := "X:\\Alan\\AnnotationTransfer\\VINCEN_PDAC_CRUK_66 - 2018-11-13 14.07.36 GM_small.svs"

	svsFile, err := Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer svsFile.Close()

	fmt.Println(svsFile)

	data, err := svsFile.IFDList[0].GetSection(0).GetData()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(hex.EncodeToString(data))
}
