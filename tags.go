package tiff

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"golang.org/x/image/tiff/lzw"
)

type TiffTagID uint16

const (
	ImageWidth                TiffTagID = 256
	ImageLength               TiffTagID = 257
	BitsPerSample             TiffTagID = 258
	Compression               TiffTagID = 259
	PhotometricInterpretation TiffTagID = 262
	Make                      TiffTagID = 271
	Model                     TiffTagID = 272
	StripOffsets              TiffTagID = 273
	SamplesPerPixel           TiffTagID = 277
	RowsPerStrip              TiffTagID = 278
	StripByteCounts           TiffTagID = 279
	XResolution               TiffTagID = 282
	YResolution               TiffTagID = 283
	PlanarConfiguration       TiffTagID = 284
	ResolutionUnit            TiffTagID = 296
	Software                  TiffTagID = 305
	DateTime                  TiffTagID = 306
	YCbCrSubSampling          TiffTagID = 530
	ReferenceBlackWhite       TiffTagID = 532
	TileWidth                 TiffTagID = 322
	TileLength                TiffTagID = 323
	TileOffsets               TiffTagID = 324
	TileByteCounts            TiffTagID = 325
	SampleFormat              TiffTagID = 339
	SMinSampleValue           TiffTagID = 340
	SMaxSampleValue           TiffTagID = 341
	NewSubFileType            TiffTagID = 254
	ImageDescription          TiffTagID = 270
	XPosition                 TiffTagID = 286
	YPosition                 TiffTagID = 287

	// Probable NDPI specific
	XOffsetFromSlideCentre TiffTagID = 65422
	YOffsetFromSlideCentre TiffTagID = 65423
	OptimisationEntries    TiffTagID = 65426
)

var tagIdMap = map[uint16]TiffTagID{
	256: ImageWidth,
	257: ImageLength,
	258: BitsPerSample,
	259: Compression,
	262: PhotometricInterpretation,
	271: Make,
	272: Model,
	273: StripOffsets,
	277: SamplesPerPixel,
	278: RowsPerStrip,
	279: StripByteCounts,
	282: XResolution,
	283: YResolution,
	284: PlanarConfiguration,
	296: ResolutionUnit,
	305: Software,
	306: DateTime,
	530: YCbCrSubSampling,
	532: ReferenceBlackWhite,
	322: TileWidth,
	323: TileLength,
	324: TileOffsets,
	325: TileByteCounts,
	339: SampleFormat,
	340: SMinSampleValue,
	341: SMaxSampleValue,
	254: NewSubFileType,
	270: ImageDescription,
	286: XPosition,
	287: YPosition,

	// NDPI specific?
	65422: XOffsetFromSlideCentre,
	65423: YOffsetFromSlideCentre,
	65426: OptimisationEntries,
}

var tagNameMap = map[TiffTagID]string{
	ImageWidth:                "ImageWidth",
	ImageLength:               "ImageLength",
	BitsPerSample:             "BitsPerSample",
	Compression:               "Compression",
	PhotometricInterpretation: "PhotometricInterpretation",
	Make:                      "Make",
	Model:                     "Model",
	StripOffsets:              "StripOffsets",
	SamplesPerPixel:           "SamplesPerPixel",
	RowsPerStrip:              "RowsPerStrip",
	StripByteCounts:           "StripByteCounts",
	XResolution:               "XResolution",
	YResolution:               "YResolution",
	PlanarConfiguration:       "PlanarConfiguration",
	ResolutionUnit:            "ResolutionUnit",
	Software:                  "Software",
	DateTime:                  "DateTime",
	YCbCrSubSampling:          "YCbCrSubSampling",
	ReferenceBlackWhite:       "ReferenceBlackWhite",
	TileWidth:                 "TileWidth",
	TileLength:                "TileLength",
	TileOffsets:               "TileOffsets",
	TileByteCounts:            "TileByteCounts",
	SampleFormat:              "SampleFormat",
	SMinSampleValue:           "SMinSampleValue",
	SMaxSampleValue:           "SMaxSampleValue",
	NewSubFileType:            "NewSubFileType",
	ImageDescription:          "ImageDescription",
	XPosition:                 "XPosition",
	YPosition:                 "YPosition",

	// NDPI Specific?
	XOffsetFromSlideCentre: "XOffsetFromSlideCentre",
	YOffsetFromSlideCentre: "YOffsetFromSlideCentre",
	OptimisationEntries:    "OptimisationEntries",
}

type DataTypeID uint16

const (
	Byte      DataTypeID = 1
	ASCII     DataTypeID = 2
	Short     DataTypeID = 3
	Long      DataTypeID = 4
	Rational  DataTypeID = 5
	SByte     DataTypeID = 6
	Undefine  DataTypeID = 7
	SShort    DataTypeID = 8
	SLong     DataTypeID = 9
	SRational DataTypeID = 10
	Float     DataTypeID = 11
	Double    DataTypeID = 12
)

var dataTypeMap = map[uint16]DataTypeID{
	1:  Byte,
	2:  ASCII,
	3:  Short,
	4:  Long,
	5:  Rational,
	6:  SByte,
	7:  Undefine,
	8:  SShort,
	9:  SLong,
	10: SRational,
	11: Float,
	12: Double,
}

var dataTypeNameMap = map[DataTypeID]string{
	Byte:      "Byte",
	ASCII:     "ASCII",
	Short:     "Short",
	Long:      "Long",
	Rational:  "Rational",
	SByte:     "SByte",
	Undefine:  "Undefine",
	SShort:    "SShort",
	SLong:     "SLong",
	SRational: "SRational",
	Float:     "Float",
	Double:    "Double",
}

type CompressionID uint16

const (
	Uncompressed CompressionID = 1
	CCIT1D       CompressionID = 2
	CCITGroup3   CompressionID = 3
	CCITGroup4   CompressionID = 4
	LZW          CompressionID = 5
	JPEG         CompressionID = 6
)

func (compressionID CompressionID) Decompress(r io.Reader) ([]byte, error) {
	var uncompressedData []byte
	var err error

	switch compressionID {
	case LZW:
		readCloser := lzw.NewReader(r, lzw.MSB, 8)
		uncompressedData, err = ioutil.ReadAll(readCloser)
		readCloser.Close()
	default:
		return nil, &FormatError{msg: "Unsupported compression scheme"}
	}

	return uncompressedData, err
}

var compressionNameMap = map[CompressionID]string{
	Uncompressed: "Uncompressed",
	CCIT1D:       "CCIT1D",
	CCITGroup3:   "CCITGroup3",
	CCITGroup4:   "CCITGroup4",
	LZW:          "LZW",
	JPEG:         "JPEG",
}

var compressionTypeMap = map[uint16]CompressionID{
	1: Uncompressed,
	2: CCIT1D,
	3: CCITGroup3,
	4: CCITGroup4,
	5: LZW,
	6: JPEG,
}

type PhotometricInterpretationID uint16

const (
	WhiteIsZero      PhotometricInterpretationID = 0
	BlackIsZero      PhotometricInterpretationID = 1
	RGB              PhotometricInterpretationID = 2
	PaletteColour    PhotometricInterpretationID = 3
	TransparencyMask PhotometricInterpretationID = 4
	CMYK             PhotometricInterpretationID = 5
	YCbCr            PhotometricInterpretationID = 6
	CIELab           PhotometricInterpretationID = 8
	ICCLab           PhotometricInterpretationID = 9
	ITULab           PhotometricInterpretationID = 10
)

var photometricInterpretationNameMap = map[PhotometricInterpretationID]string{
	WhiteIsZero:      "WhiteIsZero",
	BlackIsZero:      "BlackIsZero",
	RGB:              "RGB",
	PaletteColour:    "PaletteColour",
	TransparencyMask: "TransparencyMask",
	YCbCr:            "YCbCr",
	CIELab:           "CIELab",
	ICCLab:           "ICCLab",
	ITULab:           "ITULab",
}

var photometricInterpretationTypeMap = map[uint16]PhotometricInterpretationID{
	0:  WhiteIsZero,
	1:  BlackIsZero,
	2:  RGB,
	3:  PaletteColour,
	4:  TransparencyMask,
	5:  CMYK,
	6:  YCbCr,
	8:  CIELab,
	9:  ICCLab,
	10: ITULab,
}

type ResolutionUnitID uint16

const (
	NoUnit     ResolutionUnitID = 1
	Inch       ResolutionUnitID = 2
	Centimeter ResolutionUnitID = 3
)

var resolutionUnitNameMap = map[ResolutionUnitID]string{
	NoUnit:     "NoUnit",
	Inch:       "Inch",
	Centimeter: "Centimeter",
}

var resolutionUnitTypeMap = map[uint16]ResolutionUnitID{
	1: NoUnit,
	2: Inch,
	3: Centimeter,
}

type TiffTag interface {
	process(tiffFile *TiffFile, tagData *TiffTagData)

	GetTagID() TiffTagID
	GetType() DataTypeID
	String() string
	GetValueAsString() string
}

type baseTiffTag struct {
	TagId TiffTagID
	Type  DataTypeID
}

func (tag *baseTiffTag) processBaseTag(tagData *TiffTagData) {
	tag.TagId = tagIdMap[tagData.TagId]
	tag.Type = dataTypeMap[tagData.DataType]
}

func (tag *baseTiffTag) GetTagID() TiffTagID {
	return tag.TagId
}

func (tag *baseTiffTag) GetType() DataTypeID {
	return tag.Type
}

type ByteTiffTag struct {
	baseTiffTag

	data []byte
}

func (tag *ByteTiffTag) process(tiffFile *TiffFile, tagData *TiffTagData) {
	tag.processBaseTag(tagData)

	if tagData.DataCount <= 4 {
		a := make([]byte, 4)
		tiffFile.header.Endian.PutUint32(a, tagData.DataOffset)

		tag.data = a[:tagData.DataCount]
	} else {
		tag.data = make([]byte, tagData.DataCount)

		// TODO: Do something with the error
		startLocation, _ := tiffFile.file.Seek(0, io.SeekCurrent)
		tiffFile.file.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(tiffFile.file, tiffFile.header.Endian, &tag.data)
		// TODO: Do something with the error
		tiffFile.file.Seek(startLocation, io.SeekStart)
	}
}

func (tag *ByteTiffTag) String() string {
	return tagNameMap[tag.TagId] + ": " + tag.GetValueAsString()
}

func (tag *ByteTiffTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type ASCIITiffTag struct {
	baseTiffTag

	data string
}

func (tag *ASCIITiffTag) process(tiffFile *TiffFile, tagData *TiffTagData) {
	tag.processBaseTag(tagData)
	data := make([]byte, tagData.DataCount)

	if tagData.DataCount <= 4 {
		log.Printf("NOT IMPLEMENTED: %s\n", string(tagData.DataOffset))
		//tag.data[0] = tagData.DataOffset
	} else {
		// TODO: Do something with the error
		startLocation, _ := tiffFile.file.Seek(0, io.SeekCurrent)
		tiffFile.file.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(tiffFile.file, tiffFile.header.Endian, &data)
		// TODO: Do something with the error
		tiffFile.file.Seek(startLocation, io.SeekStart)

		tag.data = string(data)
	}
}

func (tag *ASCIITiffTag) String() string {
	return tagNameMap[tag.TagId] + ": " + tag.GetValueAsString()
}

func (tag *ASCIITiffTag) GetValueAsString() string {
	return tag.data
}

type ShortTiffTag struct {
	baseTiffTag

	data []uint16
}

func (tag *ShortTiffTag) process(tiffFile *TiffFile, tagData *TiffTagData) {
	tag.processBaseTag(tagData)
	tag.data = make([]uint16, tagData.DataCount)

	if tagData.DataCount > 2 {
		// TODO: Do something with the error
		startLocation, _ := tiffFile.file.Seek(0, io.SeekCurrent)
		tiffFile.file.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(tiffFile.file, tiffFile.header.Endian, &tag.data)
		// TODO: Do something with the error
		tiffFile.file.Seek(startLocation, io.SeekStart)
	} else {
		tag.data[0] = uint16(tagData.DataOffset & 0xffff)

		if tagData.DataCount == 2 {
			tag.data[1] = uint16(tagData.DataOffset >> 16)
		}
	}
}

func (tag *ShortTiffTag) String() string {
	return tagNameMap[tag.TagId] + ": " + tag.GetValueAsString()
}

func (tag *ShortTiffTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

// TODO: Check why this order of the member variables works and the other way around doesn't...
type RationalNumber struct {
	Denomiator uint32
	Numerator  uint32
}

func (rational *RationalNumber) GetValue() float64 {
	return float64(rational.Numerator) / float64(rational.Denomiator)
}

type RationalTiffTag struct {
	baseTiffTag

	data []RationalNumber
}

func (tag *RationalTiffTag) process(tiffFile *TiffFile, tagData *TiffTagData) {
	tag.processBaseTag(tagData)
	tag.data = make([]RationalNumber, tagData.DataCount)

	// TODO: Do something with the error
	startLocation, _ := tiffFile.file.Seek(0, io.SeekCurrent)
	tiffFile.file.Seek(int64(tagData.DataOffset), io.SeekStart)
	binary.Read(tiffFile.file, tiffFile.header.Endian, &tag.data)
	// TODO: Do something with the error
	tiffFile.file.Seek(startLocation, io.SeekStart)
}

func (tag *RationalTiffTag) String() string {
	return tagNameMap[tag.TagId] + ": " + tag.GetValueAsString()
}

func (tag *RationalTiffTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type LongTiffTag struct {
	baseTiffTag

	data []uint32
}

func (tag *LongTiffTag) process(tiffFile *TiffFile, tagData *TiffTagData) {
	tag.processBaseTag(tagData)
	tag.data = make([]uint32, tagData.DataCount)

	if tagData.DataCount == 1 {
		tag.data[0] = tagData.DataOffset
	} else {
		// TODO: Do something with the error
		startLocation, _ := tiffFile.file.Seek(0, io.SeekCurrent)
		tiffFile.file.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(tiffFile.file, tiffFile.header.Endian, &tag.data)
		// TODO: Do something with the error
		tiffFile.file.Seek(startLocation, io.SeekStart)
	}
}

func (tag *LongTiffTag) String() string {
	return tagNameMap[tag.TagId] + ": " + tag.GetValueAsString()
}

func (tag *LongTiffTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type TiffTagData struct {
	TagId      uint16 /* The tag identifier  */
	DataType   uint16 /* The scalar type of the data items  */
	DataCount  uint32 /* The number of items in the tag data  */
	DataOffset uint32 /* The byte offset to the data items  */
}

func (ifd *ImageFileDirectory) processTags() error {
	var err error
	var tags []TiffTagData
	tags = make([]TiffTagData, ifd.NumTags)

	err = binary.Read(ifd.tiffFile.file, ifd.tiffFile.header.Endian, &tags)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		tagName := tagNameMap[tagIdMap[tag.TagId]]

		if tagName == "" {
			fmt.Printf("Unknown tag id %d\n", tag.TagId)
		} else {
			dataType := dataTypeMap[tag.DataType]

			//fmt.Println(tagName + ": " + dataTypeNameMap[dataType])

			switch dataType {
			case Byte:
				var byteTag ByteTiffTag
				byteTag.process(ifd.tiffFile, &tag)

				ifd.PutTiffTag(&byteTag)
			case ASCII:
				var asciiTag ASCIITiffTag
				asciiTag.process(ifd.tiffFile, &tag)

				ifd.PutTiffTag(&asciiTag)
			case Short:
				var shortTag ShortTiffTag
				shortTag.process(ifd.tiffFile, &tag)

				ifd.PutTiffTag(&shortTag)
			case Long:
				var longTag LongTiffTag
				longTag.process(ifd.tiffFile, &tag)

				ifd.PutTiffTag(&longTag)
			case Rational:
				var rationalTag RationalTiffTag
				rationalTag.process(ifd.tiffFile, &tag)

				ifd.PutTiffTag(&rationalTag)
			default:
				fmt.Printf("Unknown tag type %s\n", dataTypeNameMap[dataTypeMap[tag.DataType]])
			}
		}
	}

	return nil
}
