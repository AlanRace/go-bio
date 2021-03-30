package test

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	tiff "github.com/AlanRace/go-bio"
)

func TestLoad(t *testing.T) {

	err := filepath.Walk("./", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match("*.tif", filepath.Base(path)); err != nil {
			return err
		} else if matched {
			log.Printf("Processing %s\n", path)
			canContinue := true
			tiffFile, err := tiff.Open(path)
			if err != nil {
				log.Printf("[ERROR] %v", err)
				canContinue = false
			}
			defer tiffFile.Close()

			if canContinue {
				//for _, ifd := range tiffFile.GetIFDList() {
				//fmt.Println(ifd)
				//}
			}
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
