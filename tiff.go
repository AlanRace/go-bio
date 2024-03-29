package gobio

import (
	"encoding/binary"
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
			log.Printf("Unknown tag id %d\n", tag.TagID)
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
				log.Printf("Unknown tag type %s\n", DataTypeNameFromID(tag.DataType))
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
		log.Printf("NOT IMPLEMENTED processASCIITag: %v\n", tagData)
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

	offsetsTag := ifd.Tags[StripOffsets]
	byteCountsTag := ifd.Tags[StripByteCounts]

	offsetsLongTag, ok := offsetsTag.(*LongTag)
	if ok {
		for index := range offsetsLongTag.Data {
			offsets = append(offsets, int64(offsetsLongTag.Data[index]))
		}
	} else {
		// Try ShortTag
		offsetsShortTag, ok := offsetsTag.(*ShortTag)

		if ok {
			for index := range offsetsShortTag.Data {
				offsets = append(offsets, int64(offsetsShortTag.Data[index]))
			}
		} else {
			return nil, nil, &FormatError{msg: "Data stored as strip, but can't convert StripOffsets to LongTag or ShortTag"}
		}
	}

	bytesCountsLongTag, ok := byteCountsTag.(*LongTag)
	if ok {
		for index := range bytesCountsLongTag.Data {
			counts = append(counts, int64(bytesCountsLongTag.Data[index]))
		}
	} else {
		// Try ShortTag
		bytesCountsShortTag, ok := byteCountsTag.(*ShortTag)

		if ok {

			for index := range bytesCountsShortTag.Data {
				counts = append(counts, int64(bytesCountsShortTag.Data[index]))
			}
		} else {
			return nil, nil, &FormatError{msg: "Data stored as strips, but can't convert StripByteCounts to LongTag or ShortTag"}
		}
	}

	return offsets, counts, nil
}

func getTileOffsets(ifd *ImageFileDirectory) ([]int64, []int64, error) {
	var offsets, counts []int64

	offsetsTag := ifd.Tags[TileOffsets]
	byteCountsTag := ifd.Tags[TileByteCounts]

	if offsetsTag == nil {
		log.Println("Data stored as tiles, but TileOffsets appear to be missing, trying to use StripOffsets")

		offsetsTag = ifd.Tags[StripOffsets]
	}
	if byteCountsTag == nil {
		log.Println("Data stored as tiles, but TileByteCounts appear to be missing, trying to use StripByteCounts")

		byteCountsTag = ifd.Tags[StripByteCounts]
	}

	tileOffsetsLongTag, ok := offsetsTag.(*LongTag)
	if ok {
		for index := range tileOffsetsLongTag.Data {
			offsets = append(offsets, int64(tileOffsetsLongTag.Data[index]))
		}
	} else {
		// Try ShortTag
		tileOffsetsShortTag, ok := offsetsTag.(*ShortTag)

		if ok {
			for index := range tileOffsetsShortTag.Data {
				offsets = append(offsets, int64(tileOffsetsShortTag.Data[index]))
			}
		} else {
			return nil, nil, &FormatError{msg: "Data stored as tiles, but can't convert TileOffsets to LongTag or ShortTag"}
		}
	}

	tileBytesCountsLongTag, ok := byteCountsTag.(*LongTag)
	if ok {
		for index := range tileBytesCountsLongTag.Data {
			counts = append(counts, int64(tileBytesCountsLongTag.Data[index]))
		}
	} else {
		// Try ShortTag
		tileBytesCountsShortTag, ok := byteCountsTag.(*ShortTag)

		if ok {

			for index := range tileBytesCountsShortTag.Data {
				counts = append(counts, int64(tileBytesCountsShortTag.Data[index]))
			}
		} else {
			return nil, nil, &FormatError{msg: "Data stored as tiles, but can't convert TileByteCounts to LongTag or ShortTag"}
		}
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
