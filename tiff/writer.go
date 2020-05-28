package tiff

// import (
// 	"encoding/binary"
// 	"io"
// 	"os"
// )

// type TiffWriter struct {
// 	file *File

// 	lastIFDOffsetOffset int64
// }

// func Create(location string) (*TiffWriter, error) {
// 	var err error
// 	var writer TiffWriter
// 	writer.file = new(File)

// 	writer.file.header.Identifier = LittleEndianMarker
// 	writer.file.header.Version = VersionMarker

// 	writer.file.header.Endian = binary.LittleEndian

// 	writer.file.file, err = os.Create(location)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = binary.Write(writer.file.file, writer.file.header.Endian, writer.file.header.Identifier)
// 	if err != nil {
// 		return nil, err
// 	}
// 	err = binary.Write(writer.file.file, writer.file.header.Endian, writer.file.header.Version)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = writer.setupForNextIFD()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &writer, nil
// }

// func (writer *TiffWriter) setupForNextIFD() error {
// 	var err error

// 	writer.lastIFDOffsetOffset, err = writer.file.file.Seek(0, io.SeekCurrent)
// 	if err != nil {
// 		return err
// 	}
// 	err = binary.Write(writer.file.file, writer.file.header.Endian, uint32(0))
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// // TODO: Write data first, tile wise, and then go back and write the IFD afterwards
// type Image interface {
// 	Width() uint32
// 	Height() uint32
// 	PixelAt(x, y uint32) []float64
// 	//NumChannels() uint16
// 	//PhotometricInterpretation() PhotometricInterpretationID
// }

// type TiledIFDWriter struct {
// 	tiffWriter *TiffWriter
// 	ifd        *ImageFileDirectory
// 	image      Image

// 	order binary.ByteOrder

// 	tileByteCounts *LongTag
// 	tileOffsets    *LongTag
// }

// func (writer *TiffWriter) getFile() *os.File {
// 	return writer.file.file
// }

// func (writer *TiffWriter) NewTiledIFD(image Image) *TiledIFDWriter {
// 	var ifdWriter TiledIFDWriter
// 	ifdWriter.tiffWriter = writer
// 	ifdWriter.ifd = new(ImageFileDirectory)
// 	ifdWriter.image = image
// 	ifdWriter.order = writer.file.header.Endian

// 	writer.file.IFDList = append(writer.file.IFDList, ifdWriter.ifd)

// 	// Add general tags about the image
// 	ifdWriter.ifd.PutTag(&LongTag{baseTag: baseTag{TagID: ImageWidth, Type: Long}, data: []uint32{image.Width()}})
// 	ifdWriter.ifd.PutTag(&LongTag{baseTag: baseTag{TagID: ImageLength, Type: Long}, data: []uint32{image.Height()}})
// 	ifdWriter.ifd.PutTag(&ShortTag{baseTag: baseTag{TagID: BitsPerSample, Type: Short}, data: []uint16{8}})
// 	// Set the default to be no compression
// 	ifdWriter.ifd.PutTag(&ShortTag{baseTag: baseTag{TagID: Compression, Type: Short}, data: []uint16{uint16(Uncompressed)}})
// 	// Set the default to be black = 0
// 	ifdWriter.ifd.PutTag(&ShortTag{baseTag: baseTag{TagID: PhotometricInterpretation, Type: Short}, data: []uint16{uint16(BlackIsZero)}})
// 	ifdWriter.ifd.PutTag(&ASCIITag{baseTag: baseTag{TagID: Software, Type: ASCII}, data: "go-bio"})

// 	// Add tiled defaults
// 	ifdWriter.ifd.PutTag(&LongTag{baseTag: baseTag{TagID: TileWidth, Type: Long}, data: []uint32{image.Width()}})
// 	ifdWriter.ifd.PutTag(&LongTag{baseTag: baseTag{TagID: TileLength, Type: Long}, data: []uint32{image.Height()}})

// 	ifdWriter.tileByteCounts = &LongTag{baseTag: baseTag{TagID: TileByteCounts, Type: Long}, data: []uint32{}}
// 	ifdWriter.tileOffsets = &LongTag{baseTag: baseTag{TagID: TileOffsets, Type: Long}, data: []uint32{}}
// 	ifdWriter.ifd.PutTag(ifdWriter.tileByteCounts)
// 	ifdWriter.ifd.PutTag(ifdWriter.tileOffsets)

// 	return &ifdWriter
// }

// func (writer *TiledIFDWriter) SetTileSize(width, height uint32) {
// 	writer.ifd.PutTag(&LongTag{baseTag: baseTag{TagID: TileWidth, Type: Long}, data: []uint32{width}})
// 	writer.ifd.PutTag(&LongTag{baseTag: baseTag{TagID: TileLength, Type: Long}, data: []uint32{height}})
// }

// func (writer *TiledIFDWriter) SetResolution(x, y float64, unit ResolutionUnitID) {
// 	xRes := NewRationalNumber(x)
// 	yRes := NewRationalNumber(y)

// 	writer.ifd.PutTag(&RationalTag{baseTag: baseTag{TagID: XResolution, Type: Rational}, data: []RationalNumber{*xRes}})
// 	writer.ifd.PutTag(&RationalTag{baseTag: baseTag{TagID: YResolution, Type: Rational}, data: []RationalNumber{*yRes}})

// 	writer.ifd.PutTag(&ShortTag{baseTag: baseTag{TagID: ResolutionUnit, Type: Short}, data: []uint16{uint16(unit)}})
// }

// func (writer *TiledIFDWriter) getFile() *os.File {
// 	return writer.tiffWriter.getFile()
// }

// func (writer *TiledIFDWriter) Write() error {
// 	var err error

// 	tileWidth := writer.ifd.GetLongTagValue(TileWidth)
// 	tileLength := writer.ifd.GetLongTagValue(TileLength)

// 	tilesAcross := (writer.image.Width() + (tileWidth - 1)) / tileWidth
// 	tilesDown := (writer.image.Height() + (tileLength - 1)) / tileLength

// 	writer.tileByteCounts.data = make([]uint32, tilesAcross*tilesDown)
// 	writer.tileOffsets.data = make([]uint32, tilesAcross*tilesDown)

// 	// TODO: Write data, update tags
// 	tileIndex := 0
// 	nullByte := byte(0)

// 	for tilesY := uint32(0); tilesY < tilesDown; tilesY++ {
// 		for tilesX := uint32(0); tilesX < tilesAcross; tilesX++ {
// 			var tileData []byte

// 			for y := (tilesY * tileLength); y < ((tilesY + 1) * tileLength); y++ {
// 				for x := (tilesX * tileWidth); x < ((tilesX + 1) * tileWidth); x++ {
// 					if y < writer.image.Height() && x < writer.image.Width() {
// 						data := writer.image.PixelAt(x, y)

// 						// TODO: Convert to byte
// 						for _, value := range data {
// 							tileData = append(tileData, byte(value))
// 						}
// 					} else {
// 						tileData = append(tileData, nullByte)
// 					}
// 				}
// 			}

// 			// TODO: Necessary compression

// 			offset, err := writer.getFile().Seek(0, io.SeekCurrent)
// 			if err != nil {
// 				return err
// 			}

// 			writer.tileByteCounts.data[tileIndex] = uint32(len(tileData))
// 			writer.tileOffsets.data[tileIndex] = uint32(offset)

// 			err = binary.Write(writer.getFile(), writer.order, tileData)
// 			if err != nil {
// 				return err
// 			}

// 			tileIndex++
// 		}
// 	}

// 	// Should be at end of currently written file, so get current location
// 	currentLocation, err := writer.getFile().Seek(0, io.SeekCurrent)
// 	if err != nil {
// 		return err
// 	}

// 	// Go back to the previous location where the next IFD offset should be stored to write the location of the new IFD
// 	_, err = writer.getFile().Seek(writer.tiffWriter.lastIFDOffsetOffset, io.SeekStart)
// 	if err != nil {
// 		return err
// 	}
// 	err = binary.Write(writer.getFile(), writer.order, uint32(currentLocation))
// 	if err != nil {
// 		return err
// 	}

// 	// Go back to where we can write the IFD header
// 	_, err = writer.getFile().Seek(currentLocation, io.SeekStart)
// 	if err != nil {
// 		return err
// 	}

// 	ifdHeaderLength := int64(6 + len(writer.ifd.Tags)*12)
// 	overflowOffset := currentLocation + ifdHeaderLength

// 	err = binary.Write(writer.getFile(), writer.order, uint16(len(writer.ifd.Tags)))
// 	if err != nil {
// 		return err
// 	}

// 	currentLocation += 2

// 	//dataLocationTagOffset := int64(0)

// 	// Make sure that tags are in accending order
// 	var tagIDs TagIDSlice
// 	for tagID := range writer.ifd.Tags {
// 		tagIDs = append(tagIDs, tagID)
// 	}
// 	tagIDs.Sort()

// 	for _, tagID := range tagIDs {
// 		//if tagID == TileOffsets || tagID == StripOffsets {
// 		//	dataLocationTagOffset = currentLocation
// 		//}
// 		tag := writer.ifd.Tags[tagID]

// 		overflowOffset, err = tag.writeTag(writer.getFile(), writer.order, overflowOffset)
// 		if err != nil {
// 			return err
// 		}

// 		// Make sure that we are in the correct location for the next tag
// 		currentLocation += 12
// 		_, err = writer.getFile().Seek(currentLocation, io.SeekStart)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	err = writer.tiffWriter.setupForNextIFD()
// 	if err != nil {
// 		return err
// 	}

// 	// TODO: Write data

// 	// TODO: Write offsets at the end? Not necessary if data written first
// 	/*	fmt.Printf("Need to set up offsets at location: %d\n", dataLocationTagOffset)
// 		_, err = writer.getFile().Seek(dataLocationTagOffset+6, io.SeekStart)
// 		if err != nil {
// 			return err
// 		}
// 		err = binary.Write(writer.getFile(), writer.order, overflowOffset)
// 		if err != nil {
// 			return err
// 		}
// 		_, err = writer.getFile().Seek(overflowOffset, io.SeekStart)
// 		if err != nil {
// 			return err
// 		}*/

// 	return nil
// }

// func (writer *TiffWriter) Close() {
// 	writer.file.Close()
// }
