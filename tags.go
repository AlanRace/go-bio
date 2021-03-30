package gobio

import (
	"fmt"
	"math"
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
	Orientation  TagID = 274
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
	274: Orientation,
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
	Orientation:               "Orientation",
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

func (tagID TagID) String() string {
	return tagNameMap[tagID]
}

func AddTag(tagID TagID, description string) {
	tagIDMap[uint16(tagID)] = tagID
	tagNameMap[tagID] = description
}

func TagFromID(tagID uint16) TagID {
	return tagIDMap[tagID]
}

func TagName(tagID TagID) string {
	return tagNameMap[tagID]
}

func TagNameFromID(tagID uint16) string {
	return tagNameMap[TagID(tagID)]
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

	Long8  DataTypeID = 16
	SLong8 DataTypeID = 17
	IFD8   DataTypeID = 18
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

	16: Long8,
	17: SLong8,
	18: IFD8,
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

	Long8:  "Long8",
	SLong8: "SLong8",
	IFD8:   "IFD8",
}

func DataTypeFromID(id uint16) DataTypeID {
	return dataTypeMap[id]
}

func DataTypeName(typeID DataTypeID) string {
	return dataTypeNameMap[typeID]
}

func DataTypeNameFromID(typeID uint16) string {
	return dataTypeNameMap[DataTypeID(typeID)]
}

type CompressionID uint16

const (
	UndefinedCompression CompressionID = 0
	Uncompressed         CompressionID = 1
	CCIT1D               CompressionID = 2
	CCITGroup3           CompressionID = 3
	CCITGroup4           CompressionID = 4
	LZW                  CompressionID = 5
	OJPEG                CompressionID = 6
	JPEG                 CompressionID = 7
	PackBits             CompressionID = 32773
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

type PredictorID uint16

const (
	PredictorNone          PredictorID = 1
	PredictorHorizontal    PredictorID = 2
	PredictorFloatingPoint PredictorID = 3
)

var predictorNameMap = map[PredictorID]string{
	PredictorNone:          "PredictorNone",
	PredictorHorizontal:    "PredictorHorizontal",
	PredictorFloatingPoint: "PredictorFloatingPoint",
}

var predictorTypeMap = map[uint16]PredictorID{
	1: PredictorNone,
	2: PredictorHorizontal,
	3: PredictorFloatingPoint,
}

type Tag interface {
	TagID() TagID
	String() string
	ValueAsString() string
	// NumItems returns the number of items stored in the tag (length of array)
	NumItems() int
}

type ByteTag struct {
	ID       TagID
	DataType DataTypeID

	Data []byte
}

func (tag ByteTag) TagID() TagID {
	return tag.ID
}

// NumItems returns the number of items stored in the tag (length of array)
func (tag ByteTag) NumItems() int {
	return len(tag.Data)
}

func (tag ByteTag) String() string {
	return tagNameMap[tag.ID] + ": " + tag.ValueAsString()
}

func (tag ByteTag) ValueAsString() string {
	return fmt.Sprint(tag.Data)
}

type ASCIITag struct {
	ID       TagID
	DataType DataTypeID

	Data string
}

func (tag ASCIITag) TagID() TagID {
	return tag.ID
}

// NumItems returns the number of items stored in the tag (length of array)
func (tag ASCIITag) NumItems() int {
	return len(tag.Data)
}

func (tag ASCIITag) String() string {
	return fmt.Sprintf("%s (%d): %s", tagNameMap[tag.ID], tag.ID, tag.ValueAsString())
}

func (tag ASCIITag) ValueAsString() string {
	return tag.Data
}

type ShortTag struct {
	ID       TagID
	DataType DataTypeID

	Data []uint16
}

func NewShortTag(tagID TagID, data []uint16) *ShortTag {
	var tag ShortTag

	tag.ID = tagID
	tag.DataType = Short

	tag.Data = data

	return &tag
}

func (tag ShortTag) TagID() TagID {
	return tag.ID
}

// NumItems returns the number of items stored in the tag (length of array)
func (tag ShortTag) NumItems() int {
	return len(tag.Data)
}

func (tag ShortTag) String() string {
	return fmt.Sprintf("%s (%d): %s", tagNameMap[tag.ID], tag.ID, tag.ValueAsString())
}

func (tag ShortTag) ValueAsString() string {
	return fmt.Sprint(tag.Data)
}

// TODO: Check why this order of the member variables works and the other way around doesn't...
type RationalNumber struct {
	Numerator   uint32
	Denominator uint32
}

func (rational *RationalNumber) Value() float64 {
	return float64(rational.Numerator) / float64(rational.Denominator)
}

func NewRationalNumber(value float64) *RationalNumber {
	var newRat RationalNumber
	//https://rosettacode.org/wiki/Convert_decimal_number_to_rational#C
	/*  a: continued fraction coefficients. */
	var a, x, d, n int64
	h := []int64{0, 1, 0}
	k := []int64{1, 0, 0}

	n = 1
	var i int

	md := int64(2 ^ 32)

	//if (md <= 1) { *denom = 1; *num = (int64_t) f; return; }

	// TODO: Don't need negative for now
	//if (f < 0) { neg = 1; f = -f; }

	for value != math.Floor(value) {
		n <<= 1
		value *= 2
	}

	d = int64(value)

	/* continued fraction and check denominator each step */
	for i = 0; i < 64; i++ {
		if n > 0 {
			a = d / n
		} else {
			a = 0
		}
		//a = n ? d / n : 0;
		if i > 0 && a == 0 {
			break
		}

		x = d
		d = n
		n = x % n

		x = a
		if k[1]*a+k[0] >= md {
			x = (md - k[0]) / k[1]
			if x*2 >= a || k[1] >= md {
				i = 65
			} else {
				break
			}
		}

		h[2] = x*h[1] + h[0]
		h[0] = h[1]
		h[1] = h[2]
		k[2] = x*k[1] + k[0]
		k[0] = k[1]
		k[1] = k[2]
	}
	//*denom = k[1];
	//*num = neg ? -h[1] : h[1];

	newRat.Denominator = uint32(k[1])
	newRat.Numerator = uint32(h[1])

	return &newRat

	/*var a, b, c, d float64
	a = 0
	b = 1
	c = value * 100
	d = 1

	maxVal := 2 ^ 32

	for int(b) < maxVal && int(d) < maxVal {
		mediant := float64(a+c) / float64(b+d)
		if value == mediant {
			if int(b)+int(d) < maxVal {
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

	return &newRat*/
}

type RationalTag struct {
	ID       TagID
	DataType DataTypeID

	Data []RationalNumber
}

func NewRationalTag(tagID TagID, data []RationalNumber) *RationalTag {
	var tag RationalTag

	tag.ID = tagID
	tag.DataType = Rational

	tag.Data = data

	return &tag
}

func (tag RationalTag) TagID() TagID {
	return tag.ID
}

// GetNumItems returns the number of items stored in the tag (length of array)
func (tag RationalTag) NumItems() int {
	return len(tag.Data)
}

func (tag RationalTag) String() string {
	return fmt.Sprintf("%s (%d): %s (%f)", tagNameMap[tag.ID], tag.ID, tag.ValueAsString(), tag.Data[0].Value())
}

func (tag RationalTag) ValueAsString() string {
	return fmt.Sprint(tag.Data)
}

type LongTag struct {
	ID       TagID
	DataType DataTypeID

	Data []uint32
}

func (tag LongTag) TagID() TagID {
	return tag.ID
}

// NumItems returns the number of items stored in the tag (length of array)
func (tag LongTag) NumItems() int {
	return len(tag.Data)
}

func (tag LongTag) String() string {
	return fmt.Sprintf("%s (%d): %s", tagNameMap[tag.ID], tag.ID, tag.ValueAsString())
}

func (tag LongTag) ValueAsString() string {
	return fmt.Sprint(tag.Data)
}

type Long8Tag struct {
	ID       TagID
	DataType DataTypeID

	Data []uint64
}

func (tag Long8Tag) TagID() TagID {
	return tag.ID
}

// NumItems returns the number of items stored in the tag (length of array)
func (tag Long8Tag) NumItems() int {
	return len(tag.Data)
}

func (tag Long8Tag) String() string {
	return fmt.Sprintf("%s (%d): %s", tagNameMap[tag.ID], tag.ID, tag.ValueAsString())
}

func (tag Long8Tag) ValueAsString() string {
	return fmt.Sprint(tag.Data)
}
