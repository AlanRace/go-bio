package tiff

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"os"
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
	// TODO: Make thread safe
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

		// In exports from ImageJ it seems like RowsPerStrip is set to 0? In this case, there is a single strip
		if dataAccess.rowsPerStrip == 0 {
			dataAccess.rowsPerStrip = dataAccess.imageLength
		}

		dataAccess.stripsInImage = dataAccess.imageLength / dataAccess.rowsPerStrip

		// Check whether we have enough strips in the image to capture full length
		if (dataAccess.stripsInImage * dataAccess.rowsPerStrip) < dataAccess.imageLength {
			dataAccess.stripsInImage++
		}

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

	//log.Printf("Found tag %v\n", tag)

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

// TODO: REMOVE ERROR FROM THIS FUNCTION
// Create structure with main tags so that don't need to pass back error on each call - can be done when parsing tags for first time
func (ifd *ImageFileDirectory) GetShortTagValue(tagID TagID) (uint16, error) {
	tag := ifd.Tags[tagID]
	var value uint16

	shortTag, ok := tag.(*ShortTag)

	if ok {
		// TODO: Decide what to do when more than 1 value
		value = shortTag.data[0]
	} else {
		// TODO: Error
		return 0, &FormatError{msg: fmt.Sprintf("Couldn't convert tag to short (%s): %v", tagID.String(), tag)}
	}

	return value, nil
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

func (ifd *ImageFileDirectory) GetResolution() (float64, float64, ResolutionUnitID, error) {
	resoutionUnit, err := ifd.GetResolutionUnit()
	if err != nil {
		return 0.0, 0.0, 0, err
	}

	return ifd.GetRationalTagValue(XResolution), ifd.GetRationalTagValue(YResolution), resoutionUnit, nil
}

func (ifd *ImageFileDirectory) GetBitsPerSample() (uint16, error) {
	return ifd.GetShortTagValue(BitsPerSample)
}

func (ifd *ImageFileDirectory) GetSamplesPerPixel() (uint16, error) {
	return ifd.GetShortTagValue(SamplesPerPixel)
}

func (ifd *ImageFileDirectory) GetCompression() (CompressionID, error) {
	compressionID, err := ifd.GetShortTagValue(Compression)
	if err != nil {
		// If no compression found, then set to the default of Uncompressed
		return Uncompressed, nil
	}

	return compressionTypeMap[compressionID], nil
}

func (ifd *ImageFileDirectory) GetResolutionUnit() (ResolutionUnitID, error) {
	resolutionUnitID, err := ifd.GetShortTagValue(ResolutionUnit)
	if err != nil {
		return 0, err
	}

	return resolutionUnitTypeMap[resolutionUnitID], nil
}

func (ifd *ImageFileDirectory) GetPhotometricInterpretation() (PhotometricInterpretationID, error) {
	photometricInterpretationID, err := ifd.GetShortTagValue(PhotometricInterpretation)
	if err != nil {
		return 0, err
	}

	return photometricInterpretationTypeMap[photometricInterpretationID], nil
}

// IsReducedResolutionImage checks whether the reduced resolution bit is set in the NewSubfileType tag
// TODO: Check the SubfileType tag as well, to support older versions
func (ifd *ImageFileDirectory) IsReducedResolutionImage() bool {
	newSubfileType := ifd.GetLongTagValue(NewSubFileType)

	if newSubfileType == 0 {
		return false
	}

	reducedImage := newSubfileType & 0x01

	return reducedImage == 1
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

type TagAccess interface {
	GetTag(tagID TagID) Tag
}
