package jpeg

import (
	"bufio"
	"fmt"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	filename := "header.bin"

	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	header, err := DecodeHeader(reader)
	if err != nil {
		fmt.Println(err)
	}

	//fmt.Println("-----")

	filename = "body.bin"

	f1, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	reader = bufio.NewReader(f1)

	data, err := header.DecodeBody(reader)
	if err != nil {
		fmt.Println(err)
	}

	//fmt.Println(data)

	f2, err := os.Create("dmp.bin")
	if err != nil {
		fmt.Println(err)
	}
	defer f2.Close()

	f2.Write(data.Data)

}
