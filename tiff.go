package tiff

import (
	"bytes"
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

	tiffFile   *TiffFile
	dataAccess DataAccess
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

	//fmt.Println(header.IFDOffset)

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

	ifd.tiffFile = tiffFile

	tiffFile.file.Seek(int64(location), io.SeekStart)

	err = binary.Read(tiffFile.file, tiffFile.header.Endian, &ifd.NumTags)
	if err != nil {
		return err
	}

	err = ifd.processTags()
	if err != nil {
		return err
	}

	err = ifd.setUpDataAccess()
	if err != nil {
		return err
	}

	err = binary.Read(tiffFile.file, tiffFile.header.Endian, &ifd.NextIFDOffset)
	if err != nil {
		return err
	}

	tiffFile.IFDList = append(tiffFile.IFDList, &ifd)

	//fmt.Println(ifd.NextIFDOffset)
	if ifd.NextIFDOffset != 0 {
		tiffFile.processIFD(ifd.NextIFDOffset)
	}

	return nil
}

func (ifd *ImageFileDirectory) setUpDataAccess() error {
	// Check if the data is tiled or stripped
	if ifd.Tags[RowsPerStrip] != nil {
		var dataAccess StripDataAccess
		ifd.dataAccess = &dataAccess

		dataAccess.tiffFile = ifd.tiffFile
		dataAccess.imageWidth, dataAccess.imageLength = ifd.GetImageDimensions()
		dataAccess.compression = ifd.GetCompression()

		dataAccess.rowsPerStrip = ifd.GetLongTagValue(RowsPerStrip)
		dataAccess.stripsInImage = (dataAccess.imageWidth * (dataAccess.rowsPerStrip - 1)) / dataAccess.rowsPerStrip

		stripOffsetsTag, ok := ifd.Tags[StripOffsets].(*LongTiffTag)
		if !ok {
			return &FormatError{msg: "Data stored as strips, but StripOffsets appear to be missing"}
		}

		stripByteCountsTag, ok := ifd.Tags[StripByteCounts].(*LongTiffTag)
		if !ok {
			return &FormatError{msg: "Data stored as strips, but StripByteCounts appear to be missing"}
		}

		dataAccess.offsets = stripOffsetsTag.data
		dataAccess.byteCounts = stripByteCountsTag.data

		return nil
	} else if ifd.Tags[TileWidth] != nil {
		var dataAccess TileDataAccess
		ifd.dataAccess = &dataAccess

		dataAccess.tiffFile = ifd.tiffFile
		dataAccess.imageWidth, dataAccess.imageLength = ifd.GetImageDimensions()
		dataAccess.compression = ifd.GetCompression()

		dataAccess.tileWidth = ifd.GetLongTagValue(TileWidth)
		dataAccess.tileLength = ifd.GetLongTagValue(TileLength)

		dataAccess.tilesAcross = (dataAccess.imageWidth + (dataAccess.tileWidth - 1)) / dataAccess.tileWidth
		dataAccess.tilesDown = (dataAccess.imageLength + (dataAccess.tileLength - 1)) / dataAccess.tileLength

		tileOffsetsTag, ok := ifd.Tags[TileOffsets].(*LongTiffTag)
		if !ok {
			return &FormatError{msg: "Data stored as tiles, but TileOffsets appear to be missing"}
		}

		tileByteCountsTag, ok := ifd.Tags[TileByteCounts].(*LongTiffTag)
		if !ok {
			return &FormatError{msg: "Data stored as tiles, but TileByteCounts appear to be missing"}
		}

		dataAccess.offsets = tileOffsetsTag.data
		dataAccess.byteCounts = tileByteCountsTag.data

		return nil
	} else {
		return &FormatError{msg: "RowsPerStrip and TileWidth metadata not present, so not sure how the data is stored"}
	}
}

func (ifd *ImageFileDirectory) PutTiffTag(tag TiffTag) {
	if ifd.Tags == nil {
		ifd.Tags = make(map[TiffTagID]TiffTag)
	}

	ifd.Tags[tag.GetTagID()] = tag
}

func (ifd *ImageFileDirectory) PrintMetadata() {
	for _, tag := range ifd.Tags {
		fmt.Println(tag.String())
	}
}

func (ifd *ImageFileDirectory) GetTag(tagId TiffTagID) TiffTag {
	return ifd.Tags[tagId]
}

func (ifd *ImageFileDirectory) GetShortTagValue(tagId TiffTagID) uint16 {
	tag := ifd.Tags[tagId]
	var value uint16

	shortTag, ok := tag.(*ShortTiffTag)

	if ok {
		// TODO: Decide what to do when more than 1 value
		value = shortTag.data[0]
	} else {
		// TODO: Error
	}

	return value
}

func (ifd *ImageFileDirectory) GetLongTagValue(tagId TiffTagID) uint32 {
	tag := ifd.Tags[tagId]
	var value uint32

	longTag, ok := tag.(*LongTiffTag)

	if ok {
		// TODO: Decide what to do when more than 1 value
		value = longTag.data[0]
	} else {
		shortTag, ok := tag.(*ShortTiffTag)

		if ok {
			value = uint32(shortTag.data[0])
		} else {
			// TODO: Error
		}
	}

	return value
}

func (ifd *ImageFileDirectory) GetImageDimensions() (uint32, uint32) {
	return ifd.GetLongTagValue(ImageWidth), ifd.GetLongTagValue(ImageLength)
}

func (ifd *ImageFileDirectory) GetCompression() CompressionID {
	compressionID := ifd.GetShortTagValue(Compression)

	return compressionTypeMap[compressionID]
}

type DataAccess interface {
	// Requests data at a specific location, returns data (which could be larger than the requested region depending on tiling/slicing)
	//GetData(rect image.Rectangle) ([]byte, image.Rectangle)

	GetDataIndexAt(x uint32, y uint32) uint32
	GetUncompressedData(index uint32) ([]byte, error)
	GetData(index uint32) ([]byte, error)
}

type baseDataAccess struct {
	tiffFile *TiffFile

	imageWidth  uint32
	imageLength uint32

	compression CompressionID

	offsets    []uint32
	byteCounts []uint32
}

func (dataAccess *baseDataAccess) GetUncompressedData(index uint32) ([]byte, error) {
	var byteData []byte
	offset := dataAccess.offsets[index]
	dataSize := dataAccess.byteCounts[index]

	byteData = make([]byte, dataSize)

	// TODO: Error checking!
	dataAccess.tiffFile.file.Seek(int64(offset), io.SeekStart)
	binary.Read(dataAccess.tiffFile.file, dataAccess.tiffFile.header.Endian, &byteData)

	return byteData, nil
}

func (dataAccess *baseDataAccess) GetData(index uint32) ([]byte, error) {
	byteData, err := dataAccess.GetUncompressedData(index)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(byteData)

	uncompressedData, err := dataAccess.compression.Decompress(r)

	return uncompressedData, nil
}

type StripDataAccess struct {
	baseDataAccess

	rowsPerStrip  uint32
	stripsInImage uint32
}

func (dataAccess *StripDataAccess) GetStripDimensions() (uint32, uint32) {
	return dataAccess.imageWidth, dataAccess.rowsPerStrip
}

func (dataAccess *StripDataAccess) GetDataIndexAt(x uint32, y uint32) uint32 {
	return y / dataAccess.rowsPerStrip
}

type TileDataAccess struct {
	baseDataAccess

	tileWidth  uint32
	tileLength uint32

	tilesAcross uint32
	tilesDown   uint32
}

func (dataAccess *TileDataAccess) GetTileDimensions() (uint32, uint32) {
	return dataAccess.tileWidth, dataAccess.tileLength
}

func (dataAccess *TileDataAccess) GetTileGrid() (uint32, uint32) {
	return dataAccess.tilesAcross, dataAccess.tilesDown
}

func (dataAccess *TileDataAccess) GetDataIndexAt(x uint32, y uint32) uint32 {
	return y*dataAccess.tilesAcross + x
}

func (dataAccess *TileDataAccess) GetTileAt(x uint32, y uint32) (uint32, uint32) {
	return x / dataAccess.tileWidth, y / dataAccess.tileLength
}

/*func (dataAccess *TileDataAccess) GetUncompressedTileData(x uint32, y uint32) ([]byte, error) {
	tileIndex := y*dataAccess.tilesAcross + x

	return dataAccess.getUncompressedData(tileIndex)
}*/

func (dataAccess *TileDataAccess) GetTileData(x uint32, y uint32) ([]byte, error) {
	tileIndex := y*dataAccess.tilesAcross + x

	return dataAccess.GetData(tileIndex)
}
