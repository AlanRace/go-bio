package qptiff

import (
	"gitea.alanrace.com/alan/go-tiff"
)

type QptiffFile struct {
	tiff.TiffFile

	overviewIFD *tiff.ImageFileDirectory
}

func Open(path string) (*QptiffFile, error) {
	var qptiffFile QptiffFile
	tiffFile, err := tiff.Open(path)

	qptiffFile.TiffFile = *tiffFile

	qptiffFile.overviewIFD = tiffFile.IFDList[0]

	return &qptiffFile, err
}
