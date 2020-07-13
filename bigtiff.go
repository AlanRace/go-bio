package gobio

import (
	"encoding/binary"
	"io"
	"log"
)

func readBigIFDOffset(reader io.Reader, endian binary.ByteOrder) (int64, error) {
	var offsetSize, empty uint16
	var offset uint64
	err := binary.Read(reader, endian, &offsetSize)
	if err != nil {
		return 0, err
	}
	err = binary.Read(reader, endian, &empty)
	if err != nil {
		return 0, err
	}
	err = binary.Read(reader, endian, &offset)
	if err != nil {
		return 0, err
	}

	return int64(offset), nil
}

func readBigIFD(seeker io.ReadSeeker, endian binary.ByteOrder, offset int64) (*ImageFileDirectory, error) {
	var ifd ImageFileDirectory
	var err error

	var numTags uint64
	var nextOffset uint64

	ifd.getStripOffsets = getBigStripOffsets
	ifd.getTileOffsets = getBigTileOffsets

	seeker.Seek(offset, io.SeekStart)
	err = binary.Read(seeker, endian, &numTags)
	if err != nil {
		return nil, err
	}
	ifd.NumTags = numTags

	//fmt.Printf("Found %d tags\n", numTags)

	err = processBigTags(&ifd, seeker, endian)
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
type bigTagData struct {
	TagID      uint16 /* The tag identifier  */
	DataType   uint16 /* The scalar type of the data items  */
	DataCount  uint64 /* The number of items in the tag data  */
	DataOffset uint64 /* The byte offset to the data items  */
}

func processBigTags(ifd *ImageFileDirectory, seeker io.ReadSeeker, endian binary.ByteOrder) error {
	var err error
	var tags []bigTagData
	tags = make([]bigTagData, ifd.NumTags)

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

			//fmt.Println(tagName + ": " + dataTypeNameMap[dataType] + "(" + strconv.Itoa(int(dataType)) + ")")

			switch dataType {
			case Byte, Undefined:
				byteTag := processBigByteTag(seeker, endian, &tag)

				ifd.PutTag(byteTag)
			case ASCII:
				asciiTag := processBigASCIITag(seeker, endian, &tag)

				ifd.PutTag(asciiTag)
			case Short:
				shortTag := processBigShortTag(seeker, endian, &tag)

				ifd.PutTag(shortTag)
			case Long:
				longTag := processBigLongTag(seeker, endian, &tag)

				ifd.PutTag(longTag)
			case Rational:
				rationalTag := processBigRationalTag(seeker, endian, &tag)

				ifd.PutTag(rationalTag)
			case Long8:
				long8Tag := processBigLong8Tag(seeker, endian, &tag)
				//fmt.Println(long8Tag)

				ifd.PutTag(long8Tag)
			default:
				log.Printf("Unknown tag type %s\n", DataTypeNameFromID(tag.DataType))
			}
		}
	}

	return nil
}

func processBigByteTag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *bigTagData) *ByteTag {
	var tag ByteTag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	if tagData.DataCount <= 8 {
		a := make([]byte, 8)
		endian.PutUint64(a, tagData.DataOffset)

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

func processBigASCIITag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *bigTagData) *ASCIITag {
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

func processBigRationalTag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *bigTagData) *RationalTag {
	var tag RationalTag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	tag.Data = make([]RationalNumber, tagData.DataCount)

	if tagData.DataCount == 1 {
		var num RationalNumber

		if endian == binary.BigEndian {
			num.Denominator = uint32(tagData.DataOffset >> 32)
			num.Numerator = uint32(tagData.DataOffset & 0xffffffff)
		} else {
			num.Numerator = uint32(tagData.DataOffset >> 32)
			num.Denominator = uint32(tagData.DataOffset & 0xffffffff)
		}

		tag.Data[0] = num
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

func processBigShortTag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *bigTagData) *ShortTag {
	// TODO: Update to make work with big (i.e. read in 4 values)
	var tag ShortTag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	tag.Data = make([]uint16, tagData.DataCount)

	if tagData.DataCount > 4 {
		// TODO: Do something with the error
		startLocation, _ := seeker.Seek(0, io.SeekCurrent)
		seeker.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(seeker, endian, &tag.Data)
		// TODO: Do something with the error
		seeker.Seek(startLocation, io.SeekStart)
	} else {

		if endian == binary.BigEndian {
			tag.Data[0] = uint16(tagData.DataOffset >> 48)

			if tagData.DataCount > 1 {
				tag.Data[1] = uint16((tagData.DataOffset >> 32) & 0xffffffffffff)
			}
			if tagData.DataCount > 2 {
				tag.Data[2] = uint16((tagData.DataOffset >> 16) & 0xffffffffffff)
			}
			if tagData.DataCount > 3 {
				tag.Data[2] = uint16(tagData.DataOffset & 0xffffffffffff)
			}
		} else {
			tag.Data[0] = uint16(tagData.DataOffset & 0xffffffffffff)

			if tagData.DataCount > 1 {
				tag.Data[1] = uint16((tagData.DataOffset >> 16) & 0xffffffffffff)
			}
			if tagData.DataCount > 2 {
				tag.Data[2] = uint16((tagData.DataOffset >> 32) & 0xffffffffffff)
			}
			if tagData.DataCount > 3 {
				tag.Data[3] = uint16(tagData.DataOffset >> 48)
			}
		}

		//fmt.Printf("Processing tag %v -> %v (%v) (%v)\n", tag.TagID, tagData.DataOffset, tag.data[0])
	}

	return &tag
}

func processBigLongTag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *bigTagData) *LongTag {
	var tag LongTag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	tag.Data = make([]uint32, tagData.DataCount)

	if tagData.DataCount > 2 {
		// TODO: Do something with the error
		startLocation, _ := seeker.Seek(0, io.SeekCurrent)
		seeker.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(seeker, endian, &tag.Data)
		// TODO: Do something with the error
		seeker.Seek(startLocation, io.SeekStart)
	} else {
		if endian == binary.BigEndian {
			tag.Data[0] = uint32(tagData.DataOffset >> 32)

			if tagData.DataCount == 2 {
				tag.Data[1] = uint32(tagData.DataOffset & 0xffffffff)
			}
		} else {
			tag.Data[0] = uint32(tagData.DataOffset & 0xffffffff)

			if tagData.DataCount == 2 {
				tag.Data[1] = uint32(tagData.DataOffset >> 32)
			}
		}

		//fmt.Printf("Processing tag %v -> %v (%v) (%v)\n", tag.TagID, tagData.DataOffset, tag.data[0])
	}

	/*if tagData.DataCount == 1 {
		tag.Data[0] = tagData.DataOffset
	} else {
		// TODO: Do something with the error
		startLocation, _ := seeker.Seek(0, io.SeekCurrent)
		seeker.Seek(int64(tagData.DataOffset), io.SeekStart)
		binary.Read(seeker, endian, &tag.Data)
		// TODO: Do something with the error
		seeker.Seek(startLocation, io.SeekStart)
	}*/

	return &tag
}

func processBigLong8Tag(seeker io.ReadSeeker, endian binary.ByteOrder, tagData *bigTagData) *Long8Tag {
	var tag Long8Tag

	tag.ID = TagFromID(tagData.TagID)
	tag.DataType = DataTypeFromID(tagData.DataType)

	tag.Data = make([]uint64, tagData.DataCount)

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

func getBigStripOffsets(ifd *ImageFileDirectory) ([]int64, []int64, error) {
	var offsets, counts []int64

	stripOffsetsTag, ok := ifd.Tags[StripOffsets].(*Long8Tag)
	if !ok {
		return nil, nil, &FormatError{msg: "Data stored as strips, but StripOffsets appear to be missing"}
	}

	stripByteCountsTag, ok := ifd.Tags[StripByteCounts].(*Long8Tag)
	if !ok {
		return nil, nil, &FormatError{msg: "Data stored as strips, but StripByteCounts appear to be missing"}
	}

	for index := range stripOffsetsTag.Data {
		offsets = append(offsets, int64(stripOffsetsTag.Data[index]))
		counts = append(counts, int64(stripByteCountsTag.Data[index]))
	}

	return offsets, counts, nil
}

func getBigTileOffsets(ifd *ImageFileDirectory) ([]int64, []int64, error) {
	var offsets, counts []int64

	// TODO: Merge this with getTileOffets
	// Can check the data type for the tag and then cast approppriately. This will allow other types to be used (if they are used?)

	tileOffsetsTag, ok := ifd.Tags[TileOffsets].(*Long8Tag)
	if !ok {
		return nil, nil, &FormatError{msg: "Data stored as tiles, but TileOffsets appear to be missing"}
	}

	tileByteCountsTag, ok := ifd.Tags[TileByteCounts].(*Long8Tag)
	if !ok {
		return nil, nil, &FormatError{msg: "Data stored as tiles, but TileByteCounts appear to be missing"}
	}

	for index := range tileOffsetsTag.Data {
		offsets = append(offsets, int64(tileOffsetsTag.Data[index]))
		counts = append(counts, int64(tileByteCountsTag.Data[index]))
	}

	return offsets, counts, nil
}
