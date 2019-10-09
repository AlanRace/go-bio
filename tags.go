package tiff

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
)

// TagID captures the ID of common Tiff tags
type TagID uint16
type TagIDSlice []TagID

// Len is the number of elements in the collection.
func (slice TagIDSlice) Len() int {
	return len(slice)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (slice TagIDSlice) Less(i, j int) bool {
	return slice[i] < slice[j]
}

// Swap swaps the elements with indexes i and j.
func (slice TagIDSlice) Swap(i, j int) {
	tmp := slice[i]
	slice[i] = slice[j]
	slice[j] = tmp
}

// Sort is a convenience method.
func (slice TagIDSlice) Sort() { sort.Sort(slice) }

const (
	NewSubFileType TagID = 254
	// ImageWidth describes the width of the image in pixels.
	ImageWidth TagID = 256
	// ImageLength describes the length (height) of the image in pixels.
	ImageLength TagID = 257
	// BitsPerSample describes the number of bits per sample. There can be multiple samples per pixel.
	BitsPerSample TagID = 258
	// Compression describes the compression scheme used to store the data.
	Compression TagID = 259
	// PhotometricInterpretation describes the way that pixel data is stored.
	PhotometricInterpretation TagID = 262
	ImageDescription          TagID = 270
	// Make describes the make of the microscope used to acquire the data.
	Make TagID = 271
	// Model describes the model of the microscope used to acquire the data.
	Model TagID = 272
	// StripOffsets describes the byte offset of the strips of data. This is only used when the tiff data is stored in strips (not tiles).
	StripOffsets TagID = 273
	// SamplesPerPixel describes the number of samples per pixel.
	SamplesPerPixel     TagID = 277
	RowsPerStrip        TagID = 278
	StripByteCounts     TagID = 279
	XResolution         TagID = 282
	YResolution         TagID = 283
	PlanarConfiguration TagID = 284
	XPosition           TagID = 286
	YPosition           TagID = 287
	ResolutionUnit      TagID = 296
	Software            TagID = 305
	DateTime            TagID = 306
	Predictor           TagID = 317
	TileWidth           TagID = 322
	TileLength          TagID = 323
	TileOffsets         TagID = 324
	TileByteCounts      TagID = 325
	SampleFormat        TagID = 339
	SMinSampleValue     TagID = 340
	SMaxSampleValue     TagID = 341
	JPEGTables          TagID = 347
	YCbCrSubSampling    TagID = 530
	ReferenceBlackWhite TagID = 532

	// Probable NDPI specific
	//XOffsetFromSlideCentre TagID = 65422
	//YOffsetFromSlideCentre TagID = 65423
	//OptimisationEntries    TagID = 65426
)

var tagIDMap = map[uint16]TagID{
	254: NewSubFileType,
	256: ImageWidth,
	257: ImageLength,
	258: BitsPerSample,
	259: Compression,
	262: PhotometricInterpretation,
	270: ImageDescription,
	271: Make,
	272: Model,
	273: StripOffsets,
	277: SamplesPerPixel,
	278: RowsPerStrip,
	279: StripByteCounts,
	282: XResolution,
	283: YResolution,
	286: XPosition,
	287: YPosition,
	284: PlanarConfiguration,
	296: ResolutionUnit,
	305: Software,
	306: DateTime,
	317: Predictor,
	322: TileWidth,
	323: TileLength,
	324: TileOffsets,
	325: TileByteCounts,
	339: SampleFormat,
	340: SMinSampleValue,
	341: SMaxSampleValue,
	347: JPEGTables,
	530: YCbCrSubSampling,
	532: ReferenceBlackWhite,

	// NDPI specific?
	//65422: XOffsetFromSlideCentre,
	//65423: YOffsetFromSlideCentre,
	//65426: OptimisationEntries,
}

var tagNameMap = map[TagID]string{
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
	JPEGTables:                "JPEGTables",
	Predictor:                 "Predictor",

	// NDPI Specific?
	//XOffsetFromSlideCentre: "XOffsetFromSlideCentre",
	//YOffsetFromSlideCentre: "YOffsetFromSlideCentre",
	//OptimisationEntries:    "OptimisationEntries",
}

type DataTypeID uint16

const (
	Byte      DataTypeID = 1
	ASCII     DataTypeID = 2
	Short     DataTypeID = 3
	Long      DataTypeID = 4
	Rational  DataTypeID = 5
	SByte     DataTypeID = 6
	Undefined DataTypeID = 7
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
	7:  Undefined,
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
	Undefined: "Undefined",
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
	OJPEG        CompressionID = 6
	JPEG         CompressionID = 7
)

func (compressionID CompressionID) String() string {
	return compressionNameMap[compressionID] + " (" + strconv.Itoa(int(compressionID)) + ")"
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

func (photometricInterpretationID PhotometricInterpretationID) String() string {
	return photometricInterpretationNameMap[photometricInterpretationID] + " (" + strconv.Itoa(int(photometricInterpretationID)) + ")"
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

type Tag interface {
	process(tiffFile *File, tagData *TagData)

	GetTagID() TagID
	GetType() DataTypeID
	String() string
	GetValueAsString() string

	writeTag(writer io.WriteSeeker, order binary.ByteOrder, overflowOffset int64) (int64, error)
}

type baseTag struct {
	TagID TagID
	Type  DataTypeID
}

func (tag *baseTag) processBaseTag(tagData *TagData) {
	tag.TagID = tagIDMap[tagData.TagID]
	tag.Type = dataTypeMap[tagData.DataType]
}

func (tag *baseTag) GetTagID() TagID {
	return tag.TagID
}

func (tag *baseTag) GetType() DataTypeID {
	return tag.Type
}

func (tag *baseTag) writeTagHeader(writer io.Writer, order binary.ByteOrder) error {
	var err error

	err = binary.Write(writer, order, tag.TagID)
	if err != nil {
		return err
	}

	err = binary.Write(writer, order, tag.Type)
	if err != nil {
		return err
	}

	return nil
}

type ByteTag struct {
	baseTag

	data []byte
}

func (tag *ByteTag) process(tiffFile *File, tagData *TagData) {
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

func (tag *ByteTag) writeTag(writer io.WriteSeeker, order binary.ByteOrder, overflowOffset int64) (int64, error) {
	var err error

	err = tag.writeTagHeader(writer, order)
	if err != nil {
		return overflowOffset, err
	}

	// Write the DataCount
	err = binary.Write(writer, order, uint32(len(tag.data)))
	if err != nil {
		return overflowOffset, err
	}

	if len(tag.data) > 4 {
		// Must use overflow
		err = binary.Write(writer, order, uint32(overflowOffset))
		if err != nil {
			return 0, err
		}

		_, err = writer.Seek(overflowOffset, io.SeekStart)
		if err != nil {
			return overflowOffset, err
		}

		err = binary.Write(writer, order, tag.data)
		if err != nil {
			return overflowOffset, err
		}

		overflowOffset, err = writer.Seek(0, io.SeekCurrent)
		if err != nil {
			return overflowOffset, err
		}
	} else {
		err = binary.Write(writer, order, tag.data)
		if err != nil {
			return overflowOffset, err
		}
	}

	return overflowOffset, nil
}

func (tag *ByteTag) String() string {
	return tagNameMap[tag.TagID] + ": " + tag.GetValueAsString()
}

func (tag *ByteTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type ASCIITag struct {
	baseTag

	data string
}

func (tag *ASCIITag) process(tiffFile *File, tagData *TagData) {
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

func (tag *ASCIITag) writeTag(writer io.WriteSeeker, order binary.ByteOrder, overflowOffset int64) (int64, error) {
	var err error

	err = tag.writeTagHeader(writer, order)
	if err != nil {
		return overflowOffset, err
	}

	data := []byte(tag.data)
	data = append(data, '\000')

	// Write the DataCount
	err = binary.Write(writer, order, uint32(len(data)))
	if err != nil {
		return overflowOffset, err
	}

	if len(data) > 4 {
		// Must use overflow
		err = binary.Write(writer, order, uint32(overflowOffset))
		if err != nil {
			return 0, err
		}

		_, err = writer.Seek(overflowOffset, io.SeekStart)
		if err != nil {
			return overflowOffset, err
		}

		err = binary.Write(writer, order, data)
		if err != nil {
			return overflowOffset, err
		}

		overflowOffset, err = writer.Seek(0, io.SeekCurrent)
		if err != nil {
			return overflowOffset, err
		}
	} else {
		err = binary.Write(writer, order, data)
		if err != nil {
			return overflowOffset, err
		}
	}

	return overflowOffset, nil
}

func (tag *ASCIITag) String() string {
	return tagNameMap[tag.TagID] + ": " + tag.GetValueAsString()
}

func (tag *ASCIITag) GetValueAsString() string {
	return tag.data
}

type ShortTag struct {
	baseTag

	data []uint16
}

func (tag *ShortTag) process(tiffFile *File, tagData *TagData) {
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

func (tag *ShortTag) writeTag(writer io.WriteSeeker, order binary.ByteOrder, overflowOffset int64) (int64, error) {
	var err error

	err = tag.writeTagHeader(writer, order)
	if err != nil {
		return overflowOffset, err
	}

	// Write the DataCount
	err = binary.Write(writer, order, uint32(len(tag.data)))
	if err != nil {
		return overflowOffset, err
	}

	if len(tag.data) > 2 {
		// Must use overflow
		err = binary.Write(writer, order, uint32(overflowOffset))
		if err != nil {
			return 0, err
		}

		_, err = writer.Seek(overflowOffset, io.SeekStart)
		if err != nil {
			return overflowOffset, err
		}

		err = binary.Write(writer, order, tag.data)
		if err != nil {
			return overflowOffset, err
		}

		overflowOffset, err = writer.Seek(0, io.SeekCurrent)
		if err != nil {
			return overflowOffset, err
		}
	} else {
		err = binary.Write(writer, order, tag.data)
		if err != nil {
			return overflowOffset, err
		}
	}

	return overflowOffset, nil
}

func (tag *ShortTag) String() string {
	return tagNameMap[tag.TagID] + ": " + tag.GetValueAsString()
}

func (tag *ShortTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

// TODO: Check why this order of the member variables works and the other way around doesn't...
type RationalNumber struct {
	Denominator uint32
	Numerator   uint32
}

func (rational *RationalNumber) GetValue() float64 {
	return float64(rational.Numerator) / float64(rational.Denominator)
}

func NewRationalNumber(value float64) *RationalNumber {
	var newRat RationalNumber
	var a, b, c, d int
	a = 0
	b = 1
	c = 1
	d = 1

	maxVal := 2 ^ 32

	for b < maxVal && d < maxVal {
		mediant := float64(a+c) / float64(b+d)
		if value == mediant {
			if b+d < maxVal {
				newRat.Numerator = uint32(a + c)
				newRat.Denominator = uint32(b + d)

				return &newRat
			} else if d > b {
				newRat.Numerator = uint32(c)
				newRat.Denominator = uint32(b)

				return &newRat
			} else {
				newRat.Numerator = uint32(a)
				newRat.Denominator = uint32(b)

				return &newRat
			}
		} else if value > mediant {
			a += c
			b += d
		} else {
			c += a
			d += b
		}
	}

	if int(b) > maxVal {
		newRat.Numerator = uint32(c)
		newRat.Denominator = uint32(d)
	} else {
		newRat.Numerator = uint32(a)
		newRat.Denominator = uint32(b)
	}

	return &newRat
}

type RationalTag struct {
	baseTag

	data []RationalNumber
}

func (tag *RationalTag) process(tiffFile *File, tagData *TagData) {
	tag.processBaseTag(tagData)
	tag.data = make([]RationalNumber, tagData.DataCount)

	// TODO: Do something with the error
	startLocation, _ := tiffFile.file.Seek(0, io.SeekCurrent)
	tiffFile.file.Seek(int64(tagData.DataOffset), io.SeekStart)
	binary.Read(tiffFile.file, tiffFile.header.Endian, &tag.data)
	// TODO: Do something with the error
	tiffFile.file.Seek(startLocation, io.SeekStart)
}

func (tag *RationalTag) writeTag(writer io.WriteSeeker, order binary.ByteOrder, overflowOffset int64) (int64, error) {
	var err error

	err = tag.writeTagHeader(writer, order)
	if err != nil {
		return overflowOffset, err
	}

	// Write the DataCount
	err = binary.Write(writer, order, uint32(len(tag.data)))
	if err != nil {
		return overflowOffset, err
	}

	// Must use overflow
	err = binary.Write(writer, order, uint32(overflowOffset))
	if err != nil {
		return 0, err
	}

	_, err = writer.Seek(overflowOffset, io.SeekStart)
	if err != nil {
		return overflowOffset, err
	}

	err = binary.Write(writer, order, tag.data)
	if err != nil {
		return overflowOffset, err
	}

	overflowOffset, err = writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return overflowOffset, err
	}

	return overflowOffset, nil
}

func (tag *RationalTag) String() string {
	return tagNameMap[tag.TagID] + ": " + tag.GetValueAsString()
}

func (tag *RationalTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type LongTag struct {
	baseTag

	data []uint32
}

func (tag *LongTag) process(tiffFile *File, tagData *TagData) {
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

func (tag *LongTag) writeTag(writer io.WriteSeeker, order binary.ByteOrder, overflowOffset int64) (int64, error) {
	var err error

	err = tag.writeTagHeader(writer, order)
	if err != nil {
		return overflowOffset, err
	}

	// Write the DataCount
	err = binary.Write(writer, order, uint32(len(tag.data)))
	if err != nil {
		return overflowOffset, err
	}

	if len(tag.data) > 1 {
		// Must use overflow
		err = binary.Write(writer, order, uint32(overflowOffset))
		if err != nil {
			return 0, err
		}

		_, err = writer.Seek(overflowOffset, io.SeekStart)
		if err != nil {
			return overflowOffset, err
		}

		err = binary.Write(writer, order, tag.data)
		if err != nil {
			return overflowOffset, err
		}

		overflowOffset, err = writer.Seek(0, io.SeekCurrent)
		if err != nil {
			return overflowOffset, err
		}
	} else {
		err = binary.Write(writer, order, tag.data)
		if err != nil {
			return overflowOffset, err
		}
	}

	return overflowOffset, nil
}

func (tag *LongTag) String() string {
	return tagNameMap[tag.TagID] + ": " + tag.GetValueAsString()
}

func (tag *LongTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

// TagData captures the details of a tag as stored in a tiff file.
type TagData struct {
	TagID      uint16 /* The tag identifier  */
	DataType   uint16 /* The scalar type of the data items  */
	DataCount  uint32 /* The number of items in the tag data  */
	DataOffset uint32 /* The byte offset to the data items  */
}

func (ifd *ImageFileDirectory) processTags() error {
	var err error
	var tags []TagData
	tags = make([]TagData, ifd.NumTags)

	err = binary.Read(ifd.tiffFile.file, ifd.tiffFile.header.Endian, &tags)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		tagName := tagNameMap[tagIDMap[tag.TagID]]

		if tagName == "" {
			fmt.Printf("Unknown tag id %d\n", tag.TagID)
		} else {
			dataType := dataTypeMap[tag.DataType]

			//fmt.Println(tagName + ": " + dataTypeNameMap[dataType])

			switch dataType {
			case Byte:
				var byteTag ByteTag
				byteTag.process(ifd.tiffFile, &tag)

				ifd.PutTag(&byteTag)
			case ASCII:
				var asciiTag ASCIITag
				asciiTag.process(ifd.tiffFile, &tag)

				ifd.PutTag(&asciiTag)
			case Short:
				var shortTag ShortTag
				shortTag.process(ifd.tiffFile, &tag)

				ifd.PutTag(&shortTag)
			case Long:
				var longTag LongTag
				longTag.process(ifd.tiffFile, &tag)

				ifd.PutTag(&longTag)
			case Rational:
				var rationalTag RationalTag
				rationalTag.process(ifd.tiffFile, &tag)

				ifd.PutTag(&rationalTag)
			case Undefined:
				var byteTag ByteTag
				byteTag.process(ifd.tiffFile, &tag)

				ifd.PutTag(&byteTag)
			default:
				fmt.Printf("Unknown tag type %s\n", dataTypeNameMap[dataTypeMap[tag.DataType]])
			}
		}
	}

	return nil
}
