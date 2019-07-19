package tiff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"os"

	"github.com/AlanRace/go-bio/jpeg"
)

const (
	LittleEndianMarker uint16 = 0x4949
	BigEndianMarker    uint16 = 0x4d4d
	VersionMarker      uint16 = 0x2a
)

type ImageFileHeader struct {
	Identifier uint16
	Version    uint16
	IFDOffset  uint32

	Endian binary.ByteOrder
}

type ImageFileDirectory struct {
	NumTags       uint16
	Tags          map[TagID]Tag
	NextIFDOffset uint32

	tiffFile   *File
	dataAccess DataAccess
}

type File struct {
	file *os.File

	header  ImageFileHeader
	IFDList []*ImageFileDirectory
}

type FormatError struct {
	msg string // description of error
	//Offset int64  // error occurred after reading Offset bytes
}

func (e *FormatError) Error() string { return e.msg }

func Open(path string) (*File, error) {
	var err error
	var tiffFile File
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

	if header.Version != VersionMarker {
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

func (tiffFile *File) Close() {
	tiffFile.file.Close()
}

func (tiffFile *File) processIFD(location uint32) error {
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
	var err error

	// Check if the data is tiled or stripped
	if ifd.Tags[RowsPerStrip] != nil {
		var dataAccess StripDataAccess
		ifd.dataAccess = &dataAccess

		err = dataAccess.initialiseDataAccess(ifd)
		if err != nil {
			return err
		}

		dataAccess.rowsPerStrip = ifd.GetLongTagValue(RowsPerStrip)
		dataAccess.stripsInImage = dataAccess.imageLength / dataAccess.rowsPerStrip

		stripOffsetsTag, ok := ifd.Tags[StripOffsets].(*LongTag)
		if !ok {
			return &FormatError{msg: "Data stored as strips, but StripOffsets appear to be missing"}
		}

		stripByteCountsTag, ok := ifd.Tags[StripByteCounts].(*LongTag)
		if !ok {
			return &FormatError{msg: "Data stored as strips, but StripByteCounts appear to be missing"}
		}

		dataAccess.offsets = stripOffsetsTag.data
		dataAccess.byteCounts = stripByteCountsTag.data

		return nil
	} else if ifd.Tags[TileWidth] != nil {
		var dataAccess TileDataAccess
		ifd.dataAccess = &dataAccess

		err = dataAccess.initialiseDataAccess(ifd)
		if err != nil {
			return err
		}

		dataAccess.tileWidth = ifd.GetLongTagValue(TileWidth)
		dataAccess.tileLength = ifd.GetLongTagValue(TileLength)

		dataAccess.tilesAcross = (dataAccess.imageWidth + (dataAccess.tileWidth - 1)) / dataAccess.tileWidth
		dataAccess.tilesDown = (dataAccess.imageLength + (dataAccess.tileLength - 1)) / dataAccess.tileLength

		tileOffsetsTag, ok := ifd.Tags[TileOffsets].(*LongTag)
		if !ok {
			return &FormatError{msg: "Data stored as tiles, but TileOffsets appear to be missing"}
		}

		tileByteCountsTag, ok := ifd.Tags[TileByteCounts].(*LongTag)
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

func (ifd *ImageFileDirectory) PutTag(tag Tag) {
	if ifd.Tags == nil {
		ifd.Tags = make(map[TagID]Tag)
	}

	ifd.Tags[tag.GetTagID()] = tag
}

func (ifd *ImageFileDirectory) PrintMetadata() {
	for _, tag := range ifd.Tags {
		fmt.Println(tag.String())
	}
}

func (ifd *ImageFileDirectory) GetTag(tagID TagID) Tag {
	return ifd.Tags[tagID]
}

func (ifd *ImageFileDirectory) GetShortTagValue(tagID TagID) uint16 {
	tag := ifd.Tags[tagID]
	var value uint16

	shortTag, ok := tag.(*ShortTag)

	if ok {
		// TODO: Decide what to do when more than 1 value
		value = shortTag.data[0]
	} else {
		// TODO: Error
	}

	return value
}

func (ifd *ImageFileDirectory) GetLongTagValue(tagID TagID) uint32 {
	tag := ifd.Tags[tagID]
	var value uint32

	longTag, ok := tag.(*LongTag)

	if ok {
		// TODO: Decide what to do when more than 1 value
		value = longTag.data[0]
	} else {
		shortTag, ok := tag.(*ShortTag)

		if ok {
			value = uint32(shortTag.data[0])
		} else {
			// TODO: Error
		}
	}

	return value
}

func (ifd *ImageFileDirectory) GetRationalTagValue(tagID TagID) float64 {
	tag := ifd.Tags[tagID]

	rationalTag, ok := tag.(*RationalTag)

	if !ok {
		// TODO: Error
	}

	// TODO: Decide what to do when more than 1 value
	return rationalTag.data[0].GetValue()
}

func (ifd *ImageFileDirectory) GetImageDimensions() (uint32, uint32) {
	return ifd.GetLongTagValue(ImageWidth), ifd.GetLongTagValue(ImageLength)
}

func (ifd *ImageFileDirectory) GetResolution() (float64, float64, ResolutionUnitID) {
	return ifd.GetRationalTagValue(XResolution), ifd.GetRationalTagValue(YResolution), ifd.GetResolutionUnit()
}

func (ifd *ImageFileDirectory) GetBitsPerSample() uint16 {
	return ifd.GetShortTagValue(BitsPerSample)
}

func (ifd *ImageFileDirectory) GetSamplesPerPixel() uint16 {
	return ifd.GetShortTagValue(SamplesPerPixel)
}

func (ifd *ImageFileDirectory) GetCompression() CompressionID {
	compressionID := ifd.GetShortTagValue(Compression)

	return compressionTypeMap[compressionID]
}

func (ifd *ImageFileDirectory) GetResolutionUnit() ResolutionUnitID {
	resolutionUnitID := ifd.GetShortTagValue(ResolutionUnit)

	return resolutionUnitTypeMap[resolutionUnitID]
}

func (ifd *ImageFileDirectory) GetPhotometricInterpretation() PhotometricInterpretationID {
	photometricInterpretationID := ifd.GetShortTagValue(PhotometricInterpretation)

	return photometricInterpretationTypeMap[photometricInterpretationID]
}

/*func (ifd *ImageFileDirectory) GetDataIndexAt(x uint32, y uint32) uint32 {
	return ifd.dataAccess.GetDataIndexAt(x, y)
}*/

func (ifd *ImageFileDirectory) GetSection(index uint32) *Section {
	return ifd.dataAccess.GetSection(index)
}

func (ifd *ImageFileDirectory) GetSectionAt(x, y uint32) *Section {
	return ifd.dataAccess.GetSectionAt(x, y)
}

func (ifd *ImageFileDirectory) GetSectionDimensions() (uint32, uint32) {
	return ifd.dataAccess.GetSectionDimensions()
}

func (ifd *ImageFileDirectory) GetSectionGrid() (uint32, uint32) {
	return ifd.dataAccess.GetSectionGrid()
}

func (ifd *ImageFileDirectory) GetCompressedData(index uint32) ([]byte, error) {
	return ifd.dataAccess.GetCompressedData(index)
}

func (ifd *ImageFileDirectory) GetData(index uint32) ([]byte, error) {
	return ifd.dataAccess.GetData(index)
}

func (ifd *ImageFileDirectory) GetFullData() ([]byte, error) {
	return ifd.dataAccess.GetFullData()
}

func (ifd *ImageFileDirectory) GetImage() (image.Image, error) {
	return ifd.dataAccess.GetImage()
}

func (ifd *ImageFileDirectory) IsTiled() bool {
	return ifd.GetTag(TileWidth) != nil
}

/*func (ifd *ImageFileDirectory) GetDataAccess() DataAccess {
	return ifd.dataAccess
}*/

type DataAccess interface {
	// Requests data at a specific location, returns data (which could be larger than the requested region depending on tiling/slicing)
	//GetData(rect image.Rectangle) ([]byte, image.Rectangle)

	// TODO: Reasses whether these are necessary with the new Section API
	//GetDataIndexAt(x uint32, y uint32) uint32
	GetCompressedData(index uint32) ([]byte, error)
	GetData(index uint32) ([]byte, error)
	GetFullData() ([]byte, error)
	GetImage() (image.Image, error)

	GetPhotometricInterpretation() PhotometricInterpretationID

	GetSection(index uint32) *Section
	GetSectionAt(x, y uint32) *Section
	GetSectionDimensions() (uint32, uint32)
	GetSectionGrid() (uint32, uint32)
}

type baseDataAccess struct {
	tiffFile *File
	ifd      *ImageFileDirectory

	imageWidth  uint32
	imageLength uint32

	compressionID             CompressionID
	photometricInterpretation PhotometricInterpretationID

	compression CompressionMethod

	bitsPerSample   []uint16
	samplesPerPixel uint16

	offsets    []uint32
	byteCounts []uint32
}

func (dataAccess *baseDataAccess) initialiseDataAccess(ifd *ImageFileDirectory) error {
	dataAccess.tiffFile = ifd.tiffFile
	dataAccess.ifd = ifd
	dataAccess.imageWidth, dataAccess.imageLength = ifd.GetImageDimensions()
	dataAccess.compressionID = ifd.GetCompression()
	dataAccess.photometricInterpretation = ifd.GetPhotometricInterpretation()
	dataAccess.samplesPerPixel = ifd.GetShortTagValue(SamplesPerPixel)

	bitsPerSampleTag, ok := ifd.Tags[BitsPerSample].(*ShortTag)
	if !ok {
		return &FormatError{msg: "BitsPerSample tag appears to be missing"}
	}

	dataAccess.bitsPerSample = bitsPerSampleTag.data

	switch dataAccess.compressionID {
	case LZW:
		dataAccess.compression = &LZWCompression{}
	case JPEG:
		if dataAccess.ifd.GetTag(JPEGTables) != nil {
			tablesTag, ok := dataAccess.ifd.GetTag(JPEGTables).(*ByteTag)
			if !ok {
				return &FormatError{msg: "JPEGTables not recorded as byte"}
			}

			r := bytes.NewReader(tablesTag.data)

			header, err := jpeg.DecodeHeader(r)
			if err != nil {
				return err
			}

			dataAccess.compression = &JPEGCompression{header: header}
		} else {
			return &FormatError{msg: "No JPEGTables tag, unsupported form of JPEG compression"}
		}
	default:
		return &FormatError{msg: "Unsupported compression scheme " + dataAccess.compressionID.String()}
	}

	return nil
}

func (dataAccess *baseDataAccess) GetPhotometricInterpretation() PhotometricInterpretationID {
	return dataAccess.photometricInterpretation
}

func (dataAccess *baseDataAccess) GetImageDimensions() (uint32, uint32) {
	return dataAccess.imageWidth, dataAccess.imageLength
}

func (dataAccess *baseDataAccess) GetCompressedData(index uint32) ([]byte, error) {
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
	var r io.Reader
	byteData, err := dataAccess.GetCompressedData(index)
	if err != nil {
		return nil, err
	}
	r = bytes.NewReader(byteData)

	return dataAccess.compression.Decompress(r)
}

func (dataAccess *baseDataAccess) PixelSizeInBytes() uint32 {
	var size uint32
	var sampleIndex uint16

	for sampleIndex = 0; sampleIndex < dataAccess.samplesPerPixel; sampleIndex++ {
		size += uint32(dataAccess.bitsPerSample[sampleIndex] / 8)
	}

	return size
}

func (dataAccess *baseDataAccess) ImageSizeInBytes() uint32 {
	return dataAccess.imageWidth * dataAccess.imageLength * dataAccess.PixelSizeInBytes()
}

func (dataAccess *baseDataAccess) createImage(fullData []byte) (image.Image, error) {
	var img image.Image

	switch dataAccess.photometricInterpretation {
	case RGB:
		rgbImg := image.NewRGBA(image.Rect(0, 0, int(dataAccess.imageWidth), int(dataAccess.imageLength)))
		data := make([]byte, len(fullData)/3*4)

		for i := 0; i < len(data)/4; i++ {
			data[i*4] = fullData[i*3]
			data[i*4+1] = fullData[i*3+1]
			data[i*4+2] = fullData[i*3+2]
			data[i*4+3] = 255
		}

		rgbImg.Pix = data

		img = rgbImg
	default:
		return nil, &FormatError{msg: "Unsupported PhotometricInterpretation: " + photometricInterpretationNameMap[dataAccess.photometricInterpretation]}
	}

	return img, nil
}

type StripDataAccess struct {
	baseDataAccess

	rowsPerStrip  uint32
	stripsInImage uint32
}

func (dataAccess *StripDataAccess) GetSectionGrid() (uint32, uint32) {
	return 1, dataAccess.stripsInImage
}

func (dataAccess *StripDataAccess) GetSectionDimensions() (uint32, uint32) {
	return dataAccess.GetStripDimensions()
}

func (dataAccess *StripDataAccess) GetStripDimensions() (uint32, uint32) {
	return dataAccess.imageWidth, dataAccess.rowsPerStrip
}

/*func (dataAccess *StripDataAccess) GetDataIndexAt(x uint32, y uint32) uint32 {
	return y / dataAccess.rowsPerStrip
}*/

func (dataAccess *StripDataAccess) GetSectionAt(x uint32, y uint32) *Section {
	return dataAccess.GetSection(y / dataAccess.rowsPerStrip)
}

func (dataAccess *StripDataAccess) GetSection(index uint32) *Section {
	if index > dataAccess.stripsInImage {
		return nil
	}

	var section Section
	section.dataAccess = dataAccess
	section.X = 0
	section.Y = index
	section.Index = index

	section.Width = dataAccess.imageWidth

	if section.Y == dataAccess.stripsInImage-1 {
		section.Height = dataAccess.imageLength % dataAccess.rowsPerStrip
	} else {
		section.Height = dataAccess.rowsPerStrip
	}

	return &section
}

/*func (dataAccess *StripDataAccess) GetStripAt(x uint32, y uint32) *Section {
	var section Section
	section.dataAccess = dataAccess
	section.sectionX = 0
	section.sectionY = y / dataAccess.rowsPerStrip
	section.sectionIndex = section.sectionY

	section.sectionWidth = dataAccess.imageWidth

	if section.sectionY == dataAccess.stripsInImage-1 {
		section.sectionHeight = dataAccess.imageLength % dataAccess.rowsPerStrip
	} else {
		section.sectionHeight = dataAccess.rowsPerStrip
	}

	return &section
}*/

func (dataAccess *StripDataAccess) GetStripInBytes() uint32 {
	return dataAccess.imageWidth * dataAccess.rowsPerStrip * dataAccess.PixelSizeInBytes()
}

func (dataAccess *StripDataAccess) GetFullData() ([]byte, error) {
	var stripIndex uint32
	fullData := make([]byte, dataAccess.ImageSizeInBytes())

	bytesPerStrip := dataAccess.GetStripInBytes()

	for stripIndex = 0; stripIndex < dataAccess.stripsInImage; stripIndex++ {
		stripData, err := dataAccess.GetData(stripIndex)

		if err != nil {
			return nil, err
		}

		copy(fullData[stripIndex*bytesPerStrip:], stripData)
	}

	return fullData, nil
}

func (dataAccess *StripDataAccess) GetImage() (image.Image, error) {
	fullData, err := dataAccess.GetFullData()
	if err != nil {
		return nil, err
	}

	return dataAccess.createImage(fullData)
}

type TileDataAccess struct {
	baseDataAccess

	tileWidth  uint32
	tileLength uint32

	tilesAcross uint32
	tilesDown   uint32
}

// Section describes a single part of an image. When the tiff file is split into strips this is one strip. When the data is split into tiles this is one tile.
type Section struct {
	dataAccess DataAccess

	Width  uint32
	Height uint32

	X uint32
	Y uint32

	Index uint32
}

func (dataAccess *TileDataAccess) GetTileDimensions() (uint32, uint32) {
	return dataAccess.tileWidth, dataAccess.tileLength
}

func (dataAccess *TileDataAccess) GetSectionGrid() (uint32, uint32) {
	return dataAccess.tilesAcross, dataAccess.tilesDown
}

func (dataAccess *TileDataAccess) GetSectionDimensions() (uint32, uint32) {
	return dataAccess.GetTileDimensions()
}

func (dataAccess *TileDataAccess) GetTileGrid() (uint32, uint32) {
	return dataAccess.tilesAcross, dataAccess.tilesDown
}

/*func (dataAccess *TileDataAccess) GetDataIndexAt(x uint32, y uint32) uint32 {
	return y*dataAccess.tilesAcross + x
}*/

func (dataAccess *TileDataAccess) GetSectionAt(x uint32, y uint32) *Section {
	index := (y/dataAccess.tileLength)*dataAccess.tilesAcross + (x / dataAccess.tileWidth)

	return dataAccess.GetSection(index)
}

func (dataAccess *TileDataAccess) GetSection(index uint32) *Section {
	var section Section
	section.dataAccess = dataAccess
	section.X = index % dataAccess.tilesAcross
	section.Y = index / dataAccess.tilesAcross

	section.Index = index

	if section.X == dataAccess.tilesAcross-1 {
		section.Width = dataAccess.imageWidth % dataAccess.tileWidth
	} else {
		section.Width = dataAccess.tileWidth
	}

	if section.Y == dataAccess.tilesDown-1 {
		section.Height = dataAccess.imageLength % dataAccess.tileLength
	} else {
		section.Height = dataAccess.tileLength
	}

	return &section
}

/*func (dataAccess *TileDataAccess) GetTileAt(x uint32, y uint32) *Section {
	var section Section
	section.dataAccess = dataAccess
	section.sectionX = x / dataAccess.tileWidth
	section.sectionY = y / dataAccess.tileLength

	section.sectionIndex = section.sectionY*dataAccess.tilesAcross + section.sectionX

	if section.sectionX == dataAccess.tilesAcross-1 {
		section.sectionWidth = dataAccess.imageWidth % dataAccess.tileWidth
	} else {
		section.sectionWidth = dataAccess.tileWidth
	}

	if section.sectionY == dataAccess.tilesDown-1 {
		section.sectionHeight = dataAccess.imageLength % dataAccess.tileLength
	} else {
		section.sectionHeight = dataAccess.tileLength
	}

	return &section
}*/

/*func (dataAccess *TileDataAccess) GetUncompressedTileData(x uint32, y uint32) ([]byte, error) {
	tileIndex := y*dataAccess.tilesAcross + x

	return dataAccess.getUncompressedData(tileIndex)
}*/

func (dataAccess *TileDataAccess) GetTileData(x uint32, y uint32) ([]byte, error) {
	tileIndex := y*dataAccess.tilesAcross + x

	return dataAccess.GetData(tileIndex)
}

func (dataAccess *TileDataAccess) GetFullData() ([]byte, error) {

	return nil, &FormatError{msg: "UNIMPLEMENTED GetFullData for TileDataAccess!!"}
}

func (dataAccess *TileDataAccess) GetImage() (image.Image, error) {
	fullData, err := dataAccess.GetFullData()
	if err != nil {
		return nil, err
	}

	return dataAccess.createImage(fullData)
}

func (section *Section) GetData() ([]byte, error) {
	return section.dataAccess.GetData(section.Index)
}

func (section *Section) GetRGBData() ([]byte, error) {
	rawData, err := section.GetData()

	if err != nil {
		return nil, err
	}

	switch section.dataAccess.GetPhotometricInterpretation() {
	case RGB:
		// TODO: check if the data is interlieved

		return rawData, nil
	default:
		return nil, &FormatError{msg: "Unsupported PhotometricInterpretation: " + photometricInterpretationNameMap[section.dataAccess.GetPhotometricInterpretation()]}
	}
}

func (section *Section) GetRGBAData() ([]byte, error) {
	rawData, err := section.GetRGBData()

	if err != nil {
		return nil, err
	}

	var rgba []byte
	rgba = make([]byte, (len(rawData)/3)*4)

	for i := 0; i < (len(rawData) / 3); i++ {
		rgba[i*4] = rawData[i*3]
		rgba[i*4+1] = rawData[i*3+1]
		rgba[i*4+2] = rawData[i*3+2]
		rgba[i*4+3] = 255
	}

	return rgba, nil
}
