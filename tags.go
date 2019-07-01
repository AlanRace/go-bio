package tiff

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
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

type TiffTag interface {
	process(tiffFile *TiffFile, tagData *TiffTagData)

	GetTagId() TiffTagID
	GetType() DataTypeID
	GetValueAsString() string
}

type BaseTiffTag struct {
	TagId TiffTagID
	Type  DataTypeID
}

func (tag *BaseTiffTag) processBaseTag(tagData *TiffTagData) {
	tag.TagId = tagIdMap[tagData.TagId]
	tag.Type = dataTypeMap[tagData.DataType]
}

func (tag *BaseTiffTag) GetTagId() TiffTagID {
	return tag.TagId
}

func (tag *BaseTiffTag) GetType() DataTypeID {
	return tag.Type
}

type ByteTiffTag struct {
	BaseTiffTag

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

func (tag *ByteTiffTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type ASCIITiffTag struct {
	BaseTiffTag

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

		
		log.Printf("%s: %s\n", tagNameMap[tag.TagId], tag.data)
	}
}

func (tag *ASCIITiffTag) GetValueAsString() string {
	return tag.data
}

type ShortTiffTag struct {
	BaseTiffTag

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
		tag.data[0] = uint16(tagData.DataOffset)

		if tagData.DataCount == 2 {
			tag.data[1] = uint16(tagData.DataOffset >> 16)
		}
	}
}

func (tag *ShortTiffTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type LongTiffTag struct {
	BaseTiffTag

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

func (tag *LongTiffTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type TiffTagData struct {
	TagId      uint16 /* The tag identifier  */
	DataType   uint16 /* The scalar type of the data items  */
	DataCount  uint32 /* The number of items in the tag data  */
	DataOffset uint32 /* The byte offset to the data items  */
}

func (ifd *ImageFileDirectory) processTags(tiffFile *TiffFile) error {
	var err error
	var tags []TiffTagData
	tags = make([]TiffTagData, ifd.NumTags)

	err = binary.Read(tiffFile.file, tiffFile.header.Endian, &tags)
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
				byteTag.process(tiffFile, &tag)

				ifd.PutTiffTag(&byteTag)
			case ASCII:
				var asciiTag ASCIITiffTag
				asciiTag.process(tiffFile, &tag)

				ifd.PutTiffTag(&asciiTag)
			case Short:
				var shortTag ShortTiffTag
				shortTag.process(tiffFile, &tag)

				ifd.PutTiffTag(&shortTag)
			case Long:
				var longTag LongTiffTag
				longTag.process(tiffFile, &tag)

				ifd.PutTiffTag(&longTag)
			default:
				fmt.Printf("Unknown tag type %s\n", dataTypeNameMap[dataTypeMap[tag.DataType]])
			}
		}
	}

	return nil
}