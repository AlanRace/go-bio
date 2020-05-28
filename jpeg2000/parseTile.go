package jpeg2000

import (
	"fmt"
	"math"
)

type packetBuf struct {
	position    uint32
	buffer      uint16
	bufferSize  uint16
	skipNextBit bool

	data []byte
}

func newDataBuf(data []byte) *packetBuf {
	return &packetBuf{data: data}
}

func (buf *packetBuf) readBits(count uint16) uint16 {
	for buf.bufferSize < count {
		b := buf.data[buf.position]
		buf.position++

		if buf.skipNextBit {
			buf.buffer = (buf.buffer << 7) | uint16(b)
			buf.bufferSize += 7
			buf.skipNextBit = false
		} else {
			buf.buffer = (buf.buffer << 8) | uint16(b)
			buf.bufferSize += 8
		}
		if b == 0xff {
			buf.skipNextBit = true
		}
	}
	buf.bufferSize -= count
	return (buf.buffer >> buf.bufferSize) & uint16((1<<count)-1)
}

func (buf *packetBuf) skipMarkerIfEqual(value uint8) bool {
	if buf.data[buf.position-1] == 0xff && buf.data[buf.position] == value {
		buf.skipBytes(1)
		return true
	} else if buf.data[buf.position] == 0xff && buf.data[buf.position+1] == value {
		buf.skipBytes(2)
		return true
	}
	return false
}

func (buf *packetBuf) skipBytes(count uint32) {
	buf.position += count
}

func (buf *packetBuf) alignToByte() {
	buf.bufferSize = 0

	if buf.skipNextBit {
		buf.position++
		buf.skipNextBit = false
	}
}
func (buf *packetBuf) readCodingpasses() uint16 {
	if buf.readBits(1) == 0 {
		return 1
	}
	if buf.readBits(1) == 0 {
		return 2
	}
	value := buf.readBits(2)
	if value < 3 {
		return value + 3
	}
	value = buf.readBits(5)
	if value < 31 {
		return value + 6
	}
	value = buf.readBits(7)
	return value + 37
}

func (header *Header) parseTilePackets(data []byte) uint32 {
	dataBuf := newDataBuf(data)

	tileIndex := header.currentTile.index
	tile := header.tiles[tileIndex]
	sopMarkerUsed := header.COD.SOPMarker
	ephMarkerUsed := header.COD.EPHMarker
	packetsIterator := tile.packetsIterator

	fmt.Println("parseTilePackets")

	for dataBuf.position < uint32(len(data)) {
		dataBuf.alignToByte()
		if sopMarkerUsed && dataBuf.skipMarkerIfEqual(0x91) {
			// Skip also marker segment length and packet sequence ID
			dataBuf.skipBytes(4)
		}

		packet := packetsIterator.nextPacket()

		if dataBuf.readBits(1) == 0x00 {
			continue
		}
		fmt.Printf("Currently %d out of %d\n", dataBuf.position, len(data))
		layerNumber := uint8(packet.layerNumber)
		//var queue = [], codeblock;
		var queue []*queueItem

		for i := 0; i < len(packet.codeblocks); i++ {
			var inclusionTree *inclusionTree
			var zeroBitPlanesTree *tagTree

			codeblock := packet.codeblocks[i]
			precinct := codeblock.precinct
			codeblockColumn := codeblock.cbx - precinct.cbxMin
			codeblockRow := codeblock.cby - precinct.cbyMin

			var codeblockIncluded, firstTimeInclusion, valueReady bool

			if codeblock.included {
				codeblockIncluded = dataBuf.readBits(1) == 0x01
			} else {
				// reading inclusion tree
				precinct = codeblock.precinct

				if precinct.inclusionTree != nil {
					inclusionTree = precinct.inclusionTree
				} else {
					// building inclusion and zero bit-planes trees
					width := precinct.cbxMax - precinct.cbxMin + 1
					height := precinct.cbyMax - precinct.cbyMin + 1

					inclusionTree = newInclusionTree(width, height, layerNumber)
					zeroBitPlanesTree = newTagTree(width, height)

					precinct.inclusionTree = inclusionTree
					precinct.zeroBitPlanesTree = zeroBitPlanesTree
				}

				if inclusionTree.reset(codeblockColumn, codeblockRow, layerNumber) {
					for true {
						if dataBuf.readBits(1) == 0x01 {
							valueReady = !inclusionTree.nextLevel()

							if valueReady {
								codeblock.included = true
								codeblockIncluded = true
								firstTimeInclusion = true
								break
							}
						} else {
							inclusionTree.incrementValue(layerNumber)
							break
						}
					}
				}
			}
			if !codeblockIncluded {
				continue
			}
			if firstTimeInclusion {
				zeroBitPlanesTree = precinct.zeroBitPlanesTree
				zeroBitPlanesTree.reset(codeblockColumn, codeblockRow)

				for true {
					if dataBuf.readBits(1) == 0x01 {
						valueReady = !zeroBitPlanesTree.nextLevel()

						if valueReady {
							break
						}
					} else {
						zeroBitPlanesTree.incrementValue()
					}
				}

				codeblock.zeroBitPlanes = zeroBitPlanesTree.value
			}
			var codingpasses = dataBuf.readCodingpasses()
			for dataBuf.readBits(1) == 0x01 {
				codeblock.Lblock++
			}
			codingpassesLog2 := uint16(math.Log2(float64(codingpasses)))
			// rounding down log2
			downVal := codingpasses < 1<<codingpassesLog2
			var bits uint16
			if downVal {
				bits = codingpassesLog2 - 1
			} else {
				bits = codingpassesLog2
			}
			bits += +codeblock.Lblock

			fmt.Printf("About to read %d bits\n", bits)

			var codedDataLength = dataBuf.readBits(bits)
			queue = append(queue, &queueItem{codeblock: codeblock, codingPasses: codingpasses, dataLength: codedDataLength})
		}
		dataBuf.alignToByte()
		if ephMarkerUsed {
			dataBuf.skipMarkerIfEqual(0x92)
		}

		fmt.Println(queue)

		for len(queue) > 0 {
			var packetItem = queue[0]
			queue = queue[1:]
			codeblock := packetItem.codeblock

			codeblock.data = append(codeblock.data, &codeblockData{data: data, start: dataBuf.position, end: dataBuf.position + uint32(packetItem.dataLength), codingPasses: packetItem.codingPasses})

			dataBuf.position += uint32(packetItem.dataLength)
		}
	}
	return dataBuf.position
}

type queueItem struct {
	codeblock    *codeblock
	codingPasses uint16
	dataLength   uint16
}

type level struct {
	width    uint32
	height   uint32
	items    []byte
	itemsSet []bool
	index    uint32
}

type inclusionTree struct {
	currentLevel int
	levels       []*level
}

func newInclusionTree(width, height uint32, defaultValue byte) *inclusionTree {
	var inclusionTree inclusionTree

	levelsLength := uint32(math.Log2(float64(maxuint32(width, height)))) + 1

	for i := uint32(0); i < levelsLength; i++ {
		var level level
		level.width = width
		level.height = height
		level.items = make([]byte, width*height)

		for j := 0; j < len(level.items); j++ {
			level.items[j] = defaultValue
		}

		inclusionTree.levels = append(inclusionTree.levels, &level)

		width = uint32(math.Ceil(float64(width) / 2.0))
		height = uint32(math.Ceil(float64(height) / 2.0))
	}

	return &inclusionTree
}

func (tree *inclusionTree) reset(i, j uint32, stopValue byte) bool {
	currentLevel := 0

	for currentLevel < len(tree.levels) {
		level := tree.levels[currentLevel]
		index := i + j*level.width
		level.index = index
		value := level.items[index]

		if value == 0xff {
			break
		}

		if value > stopValue {
			tree.currentLevel = currentLevel
			// already know about this one, propagating the value to top levels
			tree.propagateValues()
			return false
		}

		i >>= 1
		j >>= 1
		currentLevel++
	}

	tree.currentLevel = currentLevel - 1
	return true
}

func (tree *inclusionTree) incrementValue(stopValue byte) {
	level := tree.levels[tree.currentLevel]
	level.items[level.index] = stopValue + 1
	tree.propagateValues()
}

func (tree *inclusionTree) propagateValues() {
	levelIndex := tree.currentLevel
	level := tree.levels[levelIndex]
	currentValue := level.items[level.index]

	for levelIndex--; levelIndex >= 0; levelIndex-- {
		level = tree.levels[levelIndex]
		level.items[level.index] = currentValue
	}
}

func (tree *inclusionTree) nextLevel() bool {
	currentLevel := tree.currentLevel
	level := tree.levels[currentLevel]
	value := level.items[level.index]

	level.items[level.index] = 0xff
	currentLevel--

	if currentLevel < 0 {
		return false
	}

	tree.currentLevel = currentLevel
	level = tree.levels[currentLevel]
	level.items[level.index] = value
	return true
}

// Section B.10.2 Tag trees
type tagTree struct {
	currentLevel int
	levels       []*level

	value byte
}

func newTagTree(width, height uint32) *tagTree {
	var tagTree tagTree

	levelsLength := uint32(math.Log2(float64(maxuint32(width, height)))) + 1
	fmt.Printf("newTagTree: levelsLength = %d\n", levelsLength)
	for i := uint32(0); i < levelsLength; i++ {
		var level level
		level.width = width
		level.height = height

		tagTree.levels = append(tagTree.levels, &level)

		width = uint32(math.Ceil(float64(width) / 2.0))
		height = uint32(math.Ceil(float64(height) / 2.0))
	}

	return &tagTree
}

func (tree *tagTree) reset(i, j uint32) {
	currentLevel := 0
	var value byte
	var level *level
	fmt.Println("Starting reset()")
	fmt.Printf("Reset: currentLevel = %d (%d)\n", currentLevel, len(tree.levels))

	for currentLevel < len(tree.levels) {
		level = tree.levels[currentLevel]

		index := i + j*level.width

		if index < uint32(len(level.items)) && level.itemsSet[index] {
			value = level.items[index]
			break
		}

		level.index = index
		i >>= 1
		j >>= 1
		currentLevel++

		fmt.Printf("Reset: currentLevel = %d\n", currentLevel)
	}

	fmt.Printf("Reset: currentLevel = %d (%d)\n", currentLevel, len(tree.levels))
	currentLevel--
	fmt.Printf("Reset: currentLevel = %d\n", currentLevel)
	level = tree.levels[currentLevel]

	for i := uint32(0); i <= level.index; i++ {
		fmt.Printf("Creating item %d with value %d\n", i, value)
		level.items = append(level.items, 0)
		level.itemsSet = append(level.itemsSet, false)
	}

	level.items[level.index] = value
	level.itemsSet[level.index] = true
	tree.currentLevel = currentLevel
	// TODO: No replacement
	//delete this.value;
}

func (tree *tagTree) incrementValue() {
	level := tree.levels[tree.currentLevel]
	level.items[level.index]++
}

func (tree *tagTree) nextLevel() bool {
	currentLevel := tree.currentLevel
	level := tree.levels[currentLevel]
	value := level.items[level.index]

	currentLevel--
	if currentLevel < 0 {
		tree.value = value
		return false
	}

	tree.currentLevel = currentLevel
	level = tree.levels[currentLevel]

	for i := uint32(0); i <= level.index; i++ {
		level.items = append(level.items, 0)
		level.itemsSet = append(level.itemsSet, false)
	}

	level.items[level.index] = value

	return true
}
