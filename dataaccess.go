package tiff

import (
	"bytes"
	"encoding/binary"
	"image"
	"io"
)

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
	GetSamplesPerPixel() uint16

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
	var err error

	dataAccess.tiffFile = ifd.tiffFile
	dataAccess.ifd = ifd
	dataAccess.imageWidth, dataAccess.imageLength = ifd.GetImageDimensions()
	dataAccess.compressionID, err = ifd.GetCompression()
	if err != nil {
		return err
	}
	dataAccess.photometricInterpretation, err = ifd.GetPhotometricInterpretation()
	if err != nil {
		return err
	}
	dataAccess.samplesPerPixel, err = ifd.GetShortTagValue(SamplesPerPixel)
	if err != nil {
		return err
	}

	bitsPerSampleTag, ok := ifd.Tags[BitsPerSample].(*ShortTag)
	if !ok {
		return &FormatError{msg: "BitsPerSample tag appears to be missing"}
	}

	dataAccess.bitsPerSample = bitsPerSampleTag.data

	createFunction := compressionFuncMap[dataAccess.compressionID]
	if createFunction != nil {
		dataAccess.compression, err = createFunction(dataAccess)

		if err != nil {
			return err
		}
		if dataAccess.compression == nil {
			return &FormatError{msg: "Unsupported compression scheme " + dataAccess.compressionID.String() + " - missing function"}
		}
	} else {
		return &FormatError{msg: "Unsupported compression scheme " + dataAccess.compressionID.String()}
	}

	return nil
}

func (dataAccess *baseDataAccess) GetTag(tagID TagID) Tag {
	return dataAccess.ifd.GetTag(tagID)
}

func (dataAccess *baseDataAccess) GetPhotometricInterpretation() PhotometricInterpretationID {
	return dataAccess.photometricInterpretation
}

func (dataAccess *baseDataAccess) GetSamplesPerPixel() uint16 {
	return dataAccess.samplesPerPixel
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
		// TODO: check if data has 4 SamplesPerPixel

		return rawData, nil
	default:
		return nil, &FormatError{msg: "Unsupported PhotometricInterpretation: " + photometricInterpretationNameMap[section.dataAccess.GetPhotometricInterpretation()]}
	}
}

func (section *Section) GetRGBAData() ([]byte, error) {
	var rgba []byte
	var err error

	switch section.dataAccess.GetSamplesPerPixel() {
	case 3:
		rawData, err := section.GetRGBData()

		if err != nil {
			return nil, err
		}

		rgba = make([]byte, (len(rawData)/3)*4)

		for i := 0; i < (len(rawData) / 3); i++ {
			rgba[i*4] = rawData[i*3]
			rgba[i*4+1] = rawData[i*3+1]
			rgba[i*4+2] = rawData[i*3+2]
			rgba[i*4+3] = 255
		}
	case 4:
		rgba, err = section.GetData()

		if err != nil {
			return nil, err
		}
	}

	return rgba, nil
}
