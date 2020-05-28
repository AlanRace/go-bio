package gobio

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
)

//type ImageFileHeader struct {
//	Identifier uint16
//	Version    uint16
//	IFDOffset  uint32
//
//	Endian binary.ByteOrder
//}

//type ImageFileDirectory struct {
//	NumTags uint16
//	Tags          map[gobio.TagID]Tag
//	NextIFDOffset uint32

//	tiffFile   *File
//	dataAccess DataAccess
//}
func readIFDOffset(reader io.Reader, endian binary.ByteOrder) (int64, error) {
	var offset uint32
	err := binary.Read(reader, endian, &offset)
	if err != nil {
		return 0, err
	}

	return int64(offset), nil
}

func readIFD(seeker io.ReadSeeker, endian binary.ByteOrder, offset int64) (*ImageFileDirectory, error) {
	var ifd ImageFileDirectory
	var err error

	var numTags uint16
	var nextOffset uint32

	ifd.getStripOffsets = getStripOffsets
	ifd.getTileOffsets = getTileOffsets

	seeker.Seek(offset, io.SeekStart)
	err = binary.Read(seeker, endian, &numTags)
	if err != nil {
		return nil, err
	}
	ifd.NumTags = uint64(numTags)

	err = processTags(&ifd, seeker, endian)
	if err != nil {
		return nil, err
	}

	err = binary.Read(seeker, endian, &nextOffset)
	if err != nil {
		return nil, err
	}
	ifd.NextIFDOffset = int64(nextOffset)

	return &ifd, nil
}

// tagData captures the details of a tag as stored in a tiff file.
type tagData struct {
	TagID      uint16 /* The tag identifier  */
	DataType   uint16 /* The scalar type of the data items  */
	DataCount  uint32 /* The number of items in the tag data  */
	DataOffset uint32 /* The byte offset to the data items  */
}

//func (ifd *ImageFileDirectory) processTags() error {
func processTags(ifd *ImageFileDirectory, seeker io.ReadSeeker, endian binary.ByteOrder) error {
	var err error
	var tags []tagData
	tags = make([]tagData, ifd.NumTags)

	err = binary.Read(seeker, endian, &tags)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		tagName := TagNameFromID(tag.TagID) //tagNameMap[tagIDMap[tag.TagID]]

		if tagName == "" {
			fmt.Printf("Unknown tag id %d\n", tag.TagID)
		} else {
			dataType := DataTypeFromID(tag.DataType)

			//fmt.Println(tagName + ": " + dataTypeNameMap[dataType])

			switch dataType {
			case Byte, Undefined:
				byteTag := processByteTag(seeker, endian, &tag)

				ifd.PutTag(byteTag)
			case ASCII:
				asciiTag := processASCIITag(seeker, endian, &tag)

				ifd.PutTag(asciiTag)
			case Short:
				shortTag := processShortTag(seeker, endian, &tag)

				ifd.PutTag(shortTag)
			case Long:
				longTag := processLongTag(seeker, endian, &tag)

				ifd.PutTag(longTag)
			case Rational:
				rationalTag := processRationalTag(seeker, endian, &tag)

				ifd.PutTag(rationalTag)
			default:
				fmt.Printf("Unknown tag type %s\n", DataTypeNameFromID(tag.DataType))
			}
		}
	}

	return nil
}

func processByteTag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *tagData) *ByteTag {
	var tag ByteTag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	if tagData.DataCount <= 4 {
		a := make([]byte, 4)
		endian.PutUint32(a, tagData.DataOffset)

		tag.Data = a[:tagData.DataCount]
	} else {
		tag.Data = make([]byte, tagData.DataCount)

		// TODO: Do something with the error
		startLocation, _ := seeker.Seek(0, io.SeekCurrent)
		seeker.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(seeker, endian, &tag.Data)
		// TODO: Do something with the error
		seeker.Seek(startLocation, io.SeekStart)
	}

	return &tag
}

func processASCIITag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *tagData) *ASCIITag {
	var tag ASCIITag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	data := make([]byte, tagData.DataCount)

	if tagData.DataCount <= 4 {
		log.Printf("NOT IMPLEMENTED: %s\n", string(tagData.DataOffset))
		//tag.data[0] = tagData.DataOffset
	} else {
		// TODO: Do something with the error
		startLocation, _ := seeker.Seek(0, io.SeekCurrent)
		seeker.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(seeker, endian, &data)
		// TODO: Do something with the error
		seeker.Seek(startLocation, io.SeekStart)

		tag.Data = string(data)
	}

	return &tag
}

func processShortTag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *tagData) *ShortTag {
	var tag ShortTag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	tag.Data = make([]uint16, tagData.DataCount)

	if tagData.DataCount > 2 {
		// TODO: Do something with the error
		startLocation, _ := seeker.Seek(0, io.SeekCurrent)
		seeker.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(seeker, endian, &tag.Data)
		// TODO: Do something with the error
		seeker.Seek(startLocation, io.SeekStart)
	} else {
		if endian == binary.BigEndian {
			tag.Data[0] = uint16(tagData.DataOffset >> 16)

			if tagData.DataCount == 2 {
				tag.Data[1] = uint16(tagData.DataOffset & 0xffff)
			}
		} else {
			tag.Data[0] = uint16(tagData.DataOffset & 0xffff)

			if tagData.DataCount == 2 {
				tag.Data[1] = uint16(tagData.DataOffset >> 16)
			}
		}

		//fmt.Printf("Processing tag %v -> %v (%v) (%v)\n", tag.TagID, tagData.DataOffset, tag.data[0])
	}

	return &tag
}

func processRationalTag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *tagData) *RationalTag {
	var tag RationalTag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	tag.Data = make([]RationalNumber, tagData.DataCount)

	// TODO: Do something with the error
	startLocation, _ := seeker.Seek(0, io.SeekCurrent)
	seeker.Seek(int64(tagData.DataOffset), io.SeekStart)
	binary.Read(seeker, endian, &tag.Data)
	// TODO: Do something with the error
	seeker.Seek(startLocation, io.SeekStart)

	return &tag
}

func processLongTag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *tagData) *LongTag {
	var tag LongTag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	tag.Data = make([]uint32, tagData.DataCount)

	if tagData.DataCount == 1 {
		tag.Data[0] = tagData.DataOffset
	} else {
		// TODO: Do something with the error
		startLocation, _ := seeker.Seek(0, io.SeekCurrent)
		seeker.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(seeker, endian, &tag.Data)
		// TODO: Do something with the error
		seeker.Seek(startLocation, io.SeekStart)
	}

	return &tag
}

func getStripOffsets(ifd *ImageFileDirectory) ([]int64, []int64, error) {
	var offsets, counts []int64

	stripOffsetsTag, ok := ifd.Tags[StripOffsets].(*LongTag)
	if !ok {
		return nil, nil, &FormatError{msg: "Data stored as strips, but StripOffsets appear to be missing"}
	}

	stripByteCountsTag, ok := ifd.Tags[StripByteCounts].(*LongTag)
	if !ok {
		return nil, nil, &FormatError{msg: "Data stored as strips, but StripByteCounts appear to be missing"}
	}

	for index := range stripOffsetsTag.Data {
		offsets = append(offsets, int64(stripOffsetsTag.Data[index]))
		counts = append(counts, int64(stripByteCountsTag.Data[index]))
	}

	return offsets, counts, nil
}

func getTileOffsets(ifd *ImageFileDirectory) ([]int64, []int64, error) {
	var offsets, counts []int64

	tileOffsetsTag, ok := ifd.Tags[TileOffsets].(*LongTag)
	if !ok {
		return nil, nil, &FormatError{msg: "Data stored as tiles, but TileOffsets appear to be missing"}
	}

	tileByteCountsTag, ok := ifd.Tags[TileByteCounts].(*LongTag)
	if !ok {
		return nil, nil, &FormatError{msg: "Data stored as tiles, but TileByteCounts appear to be missing"}
	}

	for index := range tileOffsetsTag.Data {
		offsets = append(offsets, int64(tileOffsetsTag.Data[index]))
		counts = append(counts, int64(tileByteCountsTag.Data[index]))
	}

	return offsets, counts, nil
}

//type File struct {
//	file *os.File
//
//	header  ImageFileHeader
//	IFDList []*ImageFileDirectory
//}

// func (tiffFile *File) processIFD(location uint32) error {
// 	var ifd ImageFileDirectory
// 	var err error

// 	ifd.tiffFile = tiffFile
// 	// TODO: Make thread safe
// 	tiffFile.file.Seek(int64(location), io.SeekStart)

// 	err = binary.Read(tiffFile.file, tiffFile.header.Endian, &ifd.NumTags)
// 	if err != nil {
// 		return err
// 	}

// 	err = ifd.processTags()
// 	if err != nil {
// 		return err
// 	}

// 	err = ifd.setUpDataAccess()
// 	if err != nil {
// 		return err
// 	}

// 	err = binary.Read(tiffFile.file, tiffFile.header.Endian, &ifd.NextIFDOffset)
// 	if err != nil {
// 		return err
// 	}

// 	tiffFile.IFDList = append(tiffFile.IFDList, &ifd)

// 	//fmt.Println(ifd.NextIFDOffset)
// 	if ifd.NextIFDOffset != 0 {
// 		tiffFile.processIFD(ifd.NextIFDOffset)
// 	}

// 	return nil
// }
