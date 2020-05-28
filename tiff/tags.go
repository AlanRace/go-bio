package tiff

/*type Tag interface {
	process(tiffFile *File, tagData *tagData)

	GetTagID() TagID
	GetType() DataTypeID
	String() string
	GetValueAsString() string
	// GetNumItems returns the number of items stored in the tag (length of array)
	GetNumItems() int

	writeTag(writer io.WriteSeeker, order binary.ByteOrder, overflowOffset int64) (int64, error)
}

type baseTag struct {
	TagID gobio.TagID
	Type  gobio.DataTypeID
}

func (tag *baseTag) processBaseTag(tagData *tagData) {
	tag.TagID = tagIDMap[tagData.TagID]
	tag.Type = dataTypeMap[tagData.DataType]
}

func (tag *baseTag) GetTagID() gobio.TagID {
	return tag.TagID
}

func (tag *baseTag) GetType() gobio.DataTypeID {
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

// GetNumItems returns the number of items stored in the tag (length of array)
func (tag *ByteTag) GetNumItems() int {
	return len(tag.data)
}

func (tag *ByteTag) process(tiffFile *File, tagData *tagData) {
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

// GetNumItems returns the number of items stored in the tag (length of array)
func (tag *ASCIITag) GetNumItems() int {
	return len(tag.data)
}

func (tag *ASCIITag) process(tiffFile *File, tagData *tagData) {
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
	return fmt.Sprintf("%s (%d): %s", tagNameMap[tag.TagID], tag.TagID, tag.GetValueAsString())
}

func (tag *ASCIITag) GetValueAsString() string {
	return tag.data
}

type ShortTag struct {
	baseTag

	data []uint16
}

// GetNumItems returns the number of items stored in the tag (length of array)
func (tag *ShortTag) GetNumItems() int {
	return len(tag.data)
}

func (tag *ShortTag) process(tiffFile *File, tagData *tagData) {
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
		if tiffFile.header.Endian == binary.BigEndian {
			tag.data[0] = uint16(tagData.DataOffset >> 16)

			if tagData.DataCount == 2 {
				tag.data[1] = uint16(tagData.DataOffset & 0xffff)
			}
		} else {
			tag.data[0] = uint16(tagData.DataOffset & 0xffff)

			if tagData.DataCount == 2 {
				tag.data[1] = uint16(tagData.DataOffset >> 16)
			}
		}

		//fmt.Printf("Processing tag %v -> %v (%v) (%v)\n", tag.TagID, tagData.DataOffset, tag.data[0])
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
	return fmt.Sprintf("%s (%d): %s", tagNameMap[tag.TagID], tag.TagID, tag.GetValueAsString())
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

// GetNumItems returns the number of items stored in the tag (length of array)
func (tag *RationalTag) GetNumItems() int {
	return len(tag.data)
}

func (tag *RationalTag) process(tiffFile *File, tagData *tagData) {
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
	return fmt.Sprintf("%s (%d): %s", tagNameMap[tag.TagID], tag.TagID, tag.GetValueAsString())
}

func (tag *RationalTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}

type LongTag struct {
	baseTag

	data []uint32
}

// GetNumItems returns the number of items stored in the tag (length of array)
func (tag *LongTag) GetNumItems() int {
	return len(tag.data)
}

func (tag *LongTag) process(tiffFile *File, tagData *tagData) {
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
	return fmt.Sprintf("%s (%d): %s", tagNameMap[tag.TagID], tag.TagID, tag.GetValueAsString())
}

func (tag *LongTag) GetValueAsString() string {
	return fmt.Sprint(tag.data)
}
*/
