package tiff

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	LittleEndianMarker uint16 = 0x4949
	BigEndianMarker    uint16 = 0x4d4d
	TiffVersionMarker  uint16 = 0x2a
)

type ImageFileHeader struct {
	Identifier uint16
	Version    uint16
	IFDOffset  uint32

	Endian binary.ByteOrder
}

type ImageFileDirectory struct {
	NumTags       uint16
	Tags          map[TiffTagID]TiffTag
	NextIFDOffset uint32
}

type TiffFile struct {
	file *os.File

	header  ImageFileHeader
	IFDList []*ImageFileDirectory
}

type FormatError struct {
	msg string // description of error
	//Offset int64  // error occurred after reading Offset bytes
}

func (e *FormatError) Error() string { return e.msg }

func Open(path string) (*TiffFile, error) {
	var err error
	var tiffFile TiffFile
	header := &tiffFile.header

	tiffFile.file, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	err = binary.Read(tiffFile.file, binary.LittleEndian, &header.Identifier)
	if err != nil {
		return nil, err
	}

	//fmt.Println(tiffFile.header.Identifier & 0xff)
	//fmt.Println(tiffFile.header.Identifier >> 8)

	// Check endian
	if header.Identifier == LittleEndianMarker {
		header.Endian = binary.LittleEndian
	} else if tiffFile.header.Identifier == BigEndianMarker {
		header.Endian = binary.BigEndian
	} else {
		return nil, &FormatError{msg: "Invalid endian specified"}
	}

	err = binary.Read(tiffFile.file, header.Endian, &header.Version)
	if err != nil {
		return nil, err
	}

	if header.Version != TiffVersionMarker {
		return nil, &FormatError{msg: "Unsupported tiff version"}
	}

	err = binary.Read(tiffFile.file, header.Endian, &header.IFDOffset)
	if err != nil {
		return nil, err
	}

	fmt.Println(header.IFDOffset)

	err = tiffFile.processIFD(header.IFDOffset)
	if err != nil {
		return nil, err
	}

	return &tiffFile, nil
}

func (tiffFile *TiffFile) Close() {
	tiffFile.file.Close()
}

func (tiffFile *TiffFile) processIFD(location uint32) error {
	var ifd ImageFileDirectory
	var err error

	tiffFile.file.Seek(int64(location), io.SeekStart)

	err = binary.Read(tiffFile.file, tiffFile.header.Endian, &ifd.NumTags)
	if err != nil {
		return err
	}

	err = ifd.processTags(tiffFile)
	if err != nil {
		return err
	}

	err = binary.Read(tiffFile.file, tiffFile.header.Endian, &ifd.NextIFDOffset)
	if err != nil {
		return err
	}

	tiffFile.IFDList = append(tiffFile.IFDList, &ifd)

	fmt.Println(ifd.NextIFDOffset)
	if ifd.NextIFDOffset != 0 {
		tiffFile.processIFD(ifd.NextIFDOffset)
	}

	return nil
}

func (ifd *ImageFileDirectory) PutTiffTag(tag TiffTag) {
	if ifd.Tags == nil {
		ifd.Tags = make(map[TiffTagID]TiffTag)
	}

	ifd.Tags[tag.GetTagId()] = tag
}
