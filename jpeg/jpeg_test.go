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

	header, err := decodeHeader(reader)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("-----")

	filename = "body.bin"

	f1, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f1.Close()

	reader = bufio.NewReader(f1)

	err = decodeBody(reader, header)
	if err != nil {
		fmt.Println(err)
	}
}
