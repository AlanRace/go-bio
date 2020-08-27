package gobio

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	LittleEndianMarker uint16 = 0x4949
	BigEndianMarker    uint16 = 0x4d4d

	VersionMarker uint16 = 0x2a
	BigTiffMarker uint16 = 0x2b
)

// A format holds an image format's name, magic header and how to decode it.
/*type tiffVersion struct {
	marker uint16

	readIFDOffset func(io.Reader) int64
	readIFD       func(*os.File, binary.ByteOrder, int64) (*ImageFileDirectory, error)
}

// Formats is the list of registered formats.
var (
	formatsMu     sync.Mutex
	atomicFormats atomic.Value
)

func RegisterTiffVersion(versionMarker uint16, readIFDOffset func(io.Reader) int64, readIFD func(*os.File, binary.ByteOrder, int64) (*ImageFileDirectory, error)) {
	formatsMu.Lock()
	formats, _ := atomicFormats.Load().(map[uint16]tiffVersion)
	if formats == nil {
		formats = make(map[uint16]tiffVersion)
	}

	formats[versionMarker] = tiffVersion{marker: versionMarker, readIFDOffset: readIFDOffset, readIFD: readIFD}

	atomicFormats.Store(formats)

	formatsMu.Unlock()
}*/

type FormatError struct {
	msg string // description of error
	//Offset int64  // error occurred after reading Offset bytes
}

type File struct {
	file *os.File

	header  ImageFileHeader
	IFDList []*ImageFileDirectory
}

func (tiffFile *File) Close() {
	tiffFile.file.Close()
}

type TagAccess interface {
	GetTag(tagID TagID) Tag

	GetByteTag(tagID TagID) (*ByteTag, bool)
}

func (e *FormatError) Error() string { return e.msg }

type ImageFileHeader struct {
	Identifier uint16
	Version    uint16

	Endian binary.ByteOrder
	File   *File
}

type ImageFileDirectory struct {
	NumTags       uint64
	Tags          map[TagID]Tag
	NextIFDOffset int64

	tiffFile   *File
	dataAccess DataAccess

	getStripOffsets func(*ImageFileDirectory) ([]int64, []int64, error)
	getTileOffsets  func(*ImageFileDirectory) ([]int64, []int64, error)
}

func Open(location string) (*File, error) {
	var err error
	var tiffFile File

	tiffFile.file, err = os.Open(location)
	if err != nil {
		return nil, err
	}

	var header ImageFileHeader
	err = binary.Read(tiffFile.file, binary.LittleEndian, &header.Identifier)
	if err != nil {
		return nil, err
	}

	//fmt.Println(tiffFile.header.Identifier & 0xff)
	//fmt.Println(tiffFile.header.Identifier >> 8)

	// Check endian
	if header.Identifier == LittleEndianMarker {
		header.Endian = binary.LittleEndian
	} else if header.Identifier == BigEndianMarker {
		header.Endian = binary.BigEndian
	} else {
		return nil, &FormatError{msg: "Invalid endian specified"}
	}

	err = binary.Read(tiffFile.file, header.Endian, &header.Version)
	if err != nil {
		return nil, err
	}

	//formats, _ := atomicFormats.Load().(map[uint16]tiffVersion)

	//if version, ok := formats[header.Version]; ok {
	var ifd *ImageFileDirectory
	var offset int64

	if header.Version == VersionMarker {
		offset, err = readIFDOffset(tiffFile.file, header.Endian)
	} else if header.Version == BigTiffMarker {
		offset, err = readBigIFDOffset(tiffFile.file, header.Endian)
	} else {
		return nil, &FormatError{msg: fmt.Sprintf("Unsupported tiff version: %X", header.Version)}
	}
	if err != nil {
		return nil, err
	}

	for offset != 0 {
		if header.Version == VersionMarker {
			ifd, err = readIFD(tiffFile.file, header.Endian, offset)
		} else if header.Version == BigTiffMarker {
			ifd, err = readBigIFD(tiffFile.file, header.Endian, offset)
		}

		if err != nil {
			return nil, err
		}

		ifd.tiffFile = &tiffFile
		err = ifd.setUpDataAccess()
		if err != nil {
			return nil, err
		}

		tiffFile.IFDList = append(tiffFile.IFDList, ifd)

		offset = ifd.NextIFDOffset
	}
	//}

	/*if header.Version != VersionMarker {
		return nil, &FormatError{msg: errorMessage}
	}

	err = binary.Read(tiffFile.file, header.Endian, &header.IFDOffset)
	if err != nil {
		return nil, err
	}

	//fmt.Println(header.IFDOffset)

	err = tiffFile.processIFD(header.IFDOffset)
	if err != nil {
		return nil, err
	}*/

	return &tiffFile, nil
}

func (file File) NumReducedImages() int {
	numRes := 1

	for i := 1; i < len(file.IFDList); i++ {
		if file.IFDList[i].IsReducedResolutionImage() {
			numRes++
		}
	}

	return numRes
}

func (file File) GetReducedImage(index int) *ImageFileDirectory {
	if index == 0 {
		return file.IFDList[0]
	}

	curIndex := 0
	for i := 1; i < len(file.IFDList); i++ {
		if file.IFDList[i].IsReducedResolutionImage() {
			curIndex++
		}

		if curIndex == index {
			return file.IFDList[i]
		}
	}

	return nil
}

// func Open(path string) (File, error) {
// 	var err error

// 	tiffFile, err := os.Open(path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var header ImageFileHeader
// 	err = binary.Read(tiffFile, binary.LittleEndian, &header.Identifier)
// 	if err != nil {
// 		return nil, err
// 	}

// 	//fmt.Println(tiffFile.header.Identifier & 0xff)
// 	//fmt.Println(tiffFile.header.Identifier >> 8)

// 	// Check endian
// 	if header.Identifier == LittleEndianMarker {
// 		header.Endian = binary.LittleEndian
// 	} else if header.Identifier == BigEndianMarker {
// 		header.Endian = binary.BigEndian
// 	} else {
// 		return nil, &FormatError{msg: "Invalid endian specified"}
// 	}

// 	err = binary.Read(tiffFile, header.Endian, &header.Version)
// 	if err != nil {
// 		return nil, err
// 	}

// 	tiffFile.Close()

// 	formats, _ := atomicFormats.Load().(map[uint16]tiffVersion)

// 	if version, ok := formats[header.Version]; ok {
// 		return version.open(path)
// 	}

// 	return nil, &FormatError{msg: fmt.Sprintf("Unsupported tiff version: %X", header.Version)}
// }

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

		dataAccess.offsets, dataAccess.byteCounts, err = ifd.getStripOffsets(ifd)
		if err != nil {
			return err
		}

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

		dataAccess.offsets, dataAccess.byteCounts, err = ifd.getTileOffsets(ifd)
		if err != nil {
			return err
		}

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

	ifd.Tags[tag.TagID()] = tag
}

func (ifd *ImageFileDirectory) PrintMetadata() {
	for _, tag := range ifd.Tags {
		fmt.Println(tag.String())
	}
}

func (ifd *ImageFileDirectory) HasTag(tagID TagID) bool {
	_, ok := ifd.Tags[tagID]

	return ok
}

func (ifd *ImageFileDirectory) GetTag(tagID TagID) Tag {
	return ifd.Tags[tagID]
}

func (ifd *ImageFileDirectory) GetByteTag(tagID TagID) (*ByteTag, bool) {
	tag, ok := ifd.Tags[tagID].(*ByteTag)
	return tag, ok
}

// TODO: REMOVE ERROR FROM THIS FUNCTION
// Create structure with main tags so that don't need to pass back error on each call - can be done when parsing tags for first time
func (ifd *ImageFileDirectory) GetShortTagValue(tagID TagID) (uint16, error) {
	tag := ifd.Tags[tagID]
	var value uint16

	shortTag, ok := tag.(*ShortTag)

	if ok {
		// TODO: Decide what to do when more than 1 value
		value = shortTag.Data[0]
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
		value = longTag.Data[0]
	} else {
		shortTag, ok := tag.(*ShortTag)

		if ok {
			value = uint32(shortTag.Data[0])
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
	return rationalTag.Data[0].Value()
}

func (ifd *ImageFileDirectory) GetImageDimensions() (uint32, uint32) {
	return ifd.GetLongTagValue(ImageWidth), ifd.GetLongTagValue(ImageLength)
}

// GetResolution returns the size of a pixel in x and y and the associated unit
func (ifd *ImageFileDirectory) GetResolution() (float64, float64, ResolutionUnitID, error) {
	if !ifd.HasTag(ResolutionUnit) {
		return 0.0, 0.0, 0, fmt.Errorf("No resolution unit tag present")
	}

	resolutionUnit, err := ifd.GetResolutionUnit()
	if err != nil {
		return 0.0, 0.0, 0, err
	}

	return 1.0 / ifd.GetRationalTagValue(XResolution), 1.0 / ifd.GetRationalTagValue(YResolution), resolutionUnit, nil
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

func (ifd *ImageFileDirectory) GetPredictor() PredictorID {
	predictorID, err := ifd.GetShortTagValue(Predictor)
	if err != nil {
		return 1
	}

	return predictorTypeMap[predictorID]
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

func (ifd *ImageFileDirectory) GetSectionAt(x, y int64) *Section {
	return ifd.dataAccess.GetSectionAt(x, y)
}

func (ifd *ImageFileDirectory) GetSectionDimensions() (uint32, uint32) {
	return ifd.dataAccess.GetSectionDimensions()
}

func (ifd *ImageFileDirectory) GetSectionGrid() (uint32, uint32) {
	return ifd.dataAccess.GetSectionGrid()
}

func (ifd *ImageFileDirectory) GetCompressedData(section *Section) ([]byte, error) {
	return ifd.dataAccess.GetCompressedData(section)
}

func (ifd *ImageFileDirectory) GetData(section *Section) ([]byte, error) {
	return ifd.dataAccess.GetData(section)
}

//func (ifd *ImageFileDirectory) GetFullData() ([]byte, error) {
//	return ifd.dataAccess.GetFullData()
//}

//func (ifd *ImageFileDirectory) GetImage() (image.Image, error) {
//	return ifd.dataAccess.GetImage()
//}

func (ifd *ImageFileDirectory) IsTiled() bool {
	return ifd.GetTag(TileWidth) != nil
}

/*func (ifd *ImageFileDirectory) GetDataAccess() DataAccess {
	return ifd.dataAccess
}*/
