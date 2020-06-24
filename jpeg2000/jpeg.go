package jpeg2000

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"

	"golang.org/x/text/encoding/charmap"
)

// Implementations
//https://github.com/jai-imageio/jai-imageio-jpeg2000
//https://github.com/mozilla/pdf.js/blob/master/src/core/jpx.js

//https://github.com/mdadams/jasper
//https://github.com/uclouvain/openjpeg
//https://code.google.com/archive/p/jj2000/
//https://github.com/Unidata/jj2000

const (
	Marker uint8 = 0xff
	// Delimiting markers and marker segments
	SOC uint8 = 0x4f
	SOT uint8 = 0x90
	SOD uint8 = 0x93
	EOC uint8 = 0xd9
	// Fixed infomation marker segments
	SIZ uint8 = 0x51
	// Functional marker segments
	COD uint8 = 0x52
	COC uint8 = 0x53
	RGN uint8 = 0x5e
	QCD uint8 = 0x5c
	QCC uint8 = 0x5d
	POC uint8 = 0x5f
	// Pointer marker segments
	TLM uint8 = 0x55
	PLM uint8 = 0x57
	PLT uint8 = 0x58
	PPM uint8 = 0x60
	PPT uint8 = 0x61
	// In but stream markers and marker segments
	SOP uint8 = 0x91
	EPH uint8 = 0x92
	// Informational marker segments
	CRG uint8 = 0x63
	COM uint8 = 0x64
)

type tileHeader struct {
	index      uint16
	length     uint32
	dataSize   uint32
	partIndex  uint8
	partsCount uint8

	COD *CodingStyle
	QCD *Quantization
}

type packetIterator interface {
	nextPacket() *packet
}

type layerResolutionComponentPositionIterator struct {
	header                      *Header
	layersCount                 uint16
	componentsCount             uint16
	maxDecompositionLevelsCount uint8

	tile *tile

	l uint16
	r uint8
	i uint16
	k uint32
}

type packet struct {
	layerNumber uint16
	codeblocks  []*codeblock
}

func (iterator *layerResolutionComponentPositionIterator) nextPacket() *packet {
	// Section B.12.1.1 Layer-resolution-component-position
	for ; iterator.l < iterator.layersCount; iterator.l++ {
		for ; iterator.r <= iterator.maxDecompositionLevelsCount; iterator.r++ {
			for ; iterator.i < iterator.componentsCount; iterator.i++ {
				component := iterator.tile.components[iterator.i]

				if iterator.r > component.codingStyleParameters.NumberOfLevels {
					continue
				}

				resolution := component.resolutions[iterator.r]
				numPrecincts := resolution.precinctParameters.numPrecincts

				for iterator.k < numPrecincts {
					packet := resolution.createPacket(iterator.k, iterator.l)
					iterator.k++
					return packet
				}
				iterator.k = 0
			}
			iterator.i = 0
		}
		iterator.r = 0
	}

	log.Println("Out of packets!")

	return nil
}

func newLayerResolutionComponentPositionIterator(header *Header) *layerResolutionComponentPositionIterator {
	var iterator layerResolutionComponentPositionIterator
	iterator.header = header

	//var siz = context.SIZ;
	//var tileIndex = context.currentTile.index;
	//var tile = context.tiles[tileIndex];
	iterator.tile = header.tiles[header.currentTile.index]
	iterator.layersCount = iterator.tile.codingStyleDefaultParameters.NumberOfLayers
	iterator.componentsCount = header.Size.Csiz

	for q := uint16(0); q < iterator.componentsCount; q++ {
		iterator.maxDecompositionLevelsCount = maxuint8(iterator.maxDecompositionLevelsCount, iterator.tile.components[q].codingStyleParameters.NumberOfLevels)
	}

	return &iterator
}

type tile struct {
	tx0        uint32
	ty0        uint32
	tx1        uint32
	ty1        uint32
	width      uint32
	height     uint32
	components []*tileComponent

	codingStyleDefaultParameters *CodingStyle
	packetsIterator              packetIterator
}

type tileComponent struct {
	tcx0   uint32
	tcy0   uint32
	tcx1   uint32
	tcy1   uint32
	width  uint32
	height uint32

	quantizationParameters *Quantization
	codingStyleParameters  *CodingStyle

	subbands    []*subbandStruct
	resolutions []*resolution
}

type Header struct {
	Size       size
	Components []componentSize
	COD        CodingStyle
	QCD        Quantization

	tiles       []*tile
	currentTile *tileHeader
}

func maxuint32(x, y uint32) uint32 {
	if x > y {
		return x
	}

	return y
}

func minuint32(x, y uint32) uint32 {
	if x < y {
		return x
	}

	return y
}

func maxuint8(x, y uint8) uint8 {
	if x > y {
		return x
	}

	return y
}

func minuint8(x, y uint8) uint8 {
	if x < y {
		return x
	}

	return y
}

// Implementation modified from https://github.com/mozilla/pdf.js/blob/master/src/core/jpx.js

func (header *Header) calculateTileGrids() {
	// Section B.3 Division into tile and tile-components
	numXtiles := uint32(math.Ceil(float64(header.Size.Xsiz-header.Size.XTOsiz) / float64(header.Size.XTsiz)))
	numYtiles := uint32(math.Ceil(float64(header.Size.Ysiz-header.Size.YTOsiz) / float64(header.Size.YTsiz)))

	for q := uint32(0); q < numYtiles; q++ {
		for p := uint32(0); p < numXtiles; p++ {
			var tile tile

			tile.tx0 = maxuint32(header.Size.XTOsiz+p*header.Size.XTsiz, header.Size.XOsiz)
			tile.ty0 = maxuint32(header.Size.YTOsiz+q*header.Size.YTsiz, header.Size.YOsiz)
			tile.tx1 = maxuint32(header.Size.XTOsiz+(p+1)*header.Size.XTsiz, header.Size.Xsiz)
			tile.ty1 = maxuint32(header.Size.YTOsiz+(q+1)*header.Size.YTsiz, header.Size.Ysiz)
			tile.width = tile.tx1 - tile.tx0
			tile.height = tile.ty1 - tile.ty0

			header.tiles = append(header.tiles, &tile)
		}
	}

	for i := uint32(0); i < uint32(header.Size.Csiz); i++ {
		for j := uint32(0); j < uint32(len(header.tiles)); j++ {
			var tileComponent tileComponent

			tileComponent.tcx0 = uint32(math.Ceil(float64(header.tiles[j].tx0) / float64(header.Components[i].XRsiz)))
			tileComponent.tcy0 = uint32(math.Ceil(float64(header.tiles[j].ty0) / float64(header.Components[i].YRsiz)))
			tileComponent.tcx1 = uint32(math.Ceil(float64(header.tiles[j].tx1) / float64(header.Components[i].XRsiz)))
			tileComponent.tcy1 = uint32(math.Ceil(float64(header.tiles[j].ty1) / float64(header.Components[i].YRsiz)))
			tileComponent.width = tileComponent.tcx1 - tileComponent.tcx0
			tileComponent.height = tileComponent.tcy1 - tileComponent.tcy0

			header.tiles[j].components = append(header.tiles[j].components, &tileComponent)
		}
	}
}

func (header *Header) initialiseTile() {
	for c := 0; c < int(header.Size.Csiz); c++ {
		component := header.tiles[header.currentTile.index].components[c]

		// TODO: Handle QCC and COC
		//var qcdOrQcc =
		//  context.currentTile.QCC[c] !== undefined
		//    ? context.currentTile.QCC[c]
		//    : context.currentTile.QCD;
		component.quantizationParameters = header.currentTile.QCD
		//var codOrCoc =
		//  context.currentTile.COC[c] !== undefined
		//    ? context.currentTile.COC[c]
		//    : context.currentTile.COD;
		component.codingStyleParameters = header.currentTile.COD
	}

	header.tiles[header.currentTile.index].codingStyleDefaultParameters = header.currentTile.COD
}

type resolution struct {
	trx0 uint32
	try0 uint32
	trx1 uint32
	try1 uint32

	resLevel uint8

	subbands           []*subbandStruct
	precinctParameters *precinctParameters
}

func (res *resolution) createPacket(precinctNumber uint32, layerNumber uint16) *packet {
	var precinctCodeblocks []*codeblock
	// Section B.10.8 Order of info in packet
	subbands := res.subbands

	// sub-bands already ordered in 'LL', 'HL', 'LH', and 'HH' sequence
	for i := 0; i < len(subbands); i++ {
		codeblocks := subbands[i].codeblocks

		for j := 0; j < len(codeblocks); j++ {
			if codeblocks[j].precinctNumber != precinctNumber {
				continue
			}
			precinctCodeblocks = append(precinctCodeblocks, codeblocks[j])
		}
	}
	return &packet{layerNumber: layerNumber, codeblocks: precinctCodeblocks}
}

type subbandStruct struct {
	subbandType string
	tbx0        uint32
	tby0        uint32
	tbx1        uint32
	tby1        uint32

	resolution *resolution

	codeblockParameters *codeblockParameters
	precincts           []*precinctStruct
	codeblocks          []*codeblock
}

type blockDimensions struct {
	PPx uint8
	PPy uint8

	xcb uint8
	ycb uint8
}

func (header *Header) getBlocksDimensions(component *tileComponent, resolutionLevel uint8) *blockDimensions {
	codOrCoc := component.codingStyleParameters
	var result blockDimensions

	if !codOrCoc.EntropyCoder {
		result.PPx = 15
		result.PPy = 15
	} else {
		log.Printf("getBlocksDimensions: EntropyCoder not suuported")
		//result.PPx = codOrCoc. precinctsSizes[r].PPx;
		//result.PPy = codOrCoc.precinctsSizes[r].PPy;
	}
	// calculate codeblock size as described in section B.7
	if resolutionLevel > 0 {
		result.xcb = minuint8(codOrCoc.Xcb, result.PPx-1)
		result.ycb = minuint8(codOrCoc.Ycb, result.PPy-1)
	} else {
		result.xcb = minuint8(codOrCoc.Xcb, result.PPx)
		result.ycb = minuint8(codOrCoc.Ycb, result.PPy)
	}

	return &result
}

type precinctParameters struct {
	precinctWidth           uint32
	precinctHeight          uint32
	numPrecinctsWide        uint32
	numPrecinctsHigh        uint32
	numPrecincts            uint32
	precinctWidthInSubband  uint32
	precinctHeightInSubband uint32
}

func (res *resolution) buildPrecincts(header *Header, blockDims *blockDimensions) {
	var precinctParameters precinctParameters

	// Section B.6 Division resolution to precincts
	precinctParameters.precinctWidth = 1 << blockDims.PPx
	precinctParameters.precinctHeight = 1 << blockDims.PPy

	// Jasper introduces codeblock groups for mapping each subband codeblocks
	// to precincts. Precinct partition divides a resolution according to width
	// and height parameters. The subband that belongs to the resolution level
	// has a different size than the level, unless it is the zero resolution.

	// From Jasper documentation: jpeg2000.pdf, section K: Tier-2 coding:
	// The precinct partitioning for a particular subband is derived from a
	// partitioning of its parent LL band (i.e., the LL band at the next higher
	// resolution level)... The LL band associated with each resolution level is
	// divided into precincts... Each of the resulting precinct regions is then
	// mapped into its child subbands (if any) at the next lower resolution
	// level. This is accomplished by using the coordinate transformation
	// (u, v) = (ceil(x/2), ceil(y/2)) where (x, y) and (u, v) are the
	// coordinates of a point in the LL band and child subband, respectively.
	var toAdd int32

	if res.resLevel == 0 {
		toAdd = 0
	} else {
		toAdd = -1
	}

	precinctParameters.precinctWidthInSubband = 1 << uint32(int32(blockDims.PPx)+toAdd)
	precinctParameters.precinctHeightInSubband = 1 << uint32(int32(blockDims.PPy)+toAdd)

	if res.trx1 > res.trx0 {
		precinctParameters.numPrecinctsWide = uint32(math.Ceil(float64(res.trx1)/float64(precinctParameters.precinctWidth)) - math.Floor(float64(res.trx0)/float64(precinctParameters.precinctWidth)))
	}

	if res.try1 > res.try0 {
		precinctParameters.numPrecinctsHigh = uint32(math.Ceil(float64(res.try1)/float64(precinctParameters.precinctHeight)) - math.Floor(float64(res.try0)/float64(precinctParameters.precinctHeight)))
	}

	precinctParameters.numPrecincts = precinctParameters.numPrecinctsWide * precinctParameters.numPrecinctsHigh

	res.precinctParameters = &precinctParameters
}

type codeblock struct {
	cbx  uint32
	cby  uint32
	tbx0 uint32
	tby0 uint32
	tbx1 uint32
	tby1 uint32

	tbx0_ uint32
	tby0_ uint32
	tbx1_ uint32
	tby1_ uint32

	precinct       *precinctStruct
	precinctNumber uint32
	subbandType    string
	Lblock         uint16

	data []*codeblockData

	included bool

	zeroBitPlanes byte
}

type codeblockData struct {
	data         []byte
	start        uint32
	end          uint32
	codingPasses uint16
}

type codeblockParameters struct {
	codeblockWidth   uint8
	codeblockHeight  uint8
	numcodeblockwide uint32
	numcodeblockhigh uint32
}

type precinctStruct struct {
	cbxMin uint32
	cbyMin uint32
	cbxMax uint32
	cbyMax uint32

	inclusionTree     *inclusionTree
	zeroBitPlanesTree *tagTree
}

func (subband *subbandStruct) buildCodeblocks(header *Header, blockDims *blockDimensions) {
	// Section B.7 Division sub-band into code-blocks
	xcb := blockDims.xcb
	ycb := blockDims.ycb
	codeblockWidth := uint32(1 << xcb)
	codeblockHeight := uint32(1 << ycb)
	cbx0 := subband.tbx0 >> xcb
	cby0 := subband.tby0 >> ycb
	cbx1 := (subband.tbx1 + codeblockWidth - 1) >> xcb
	cby1 := (subband.tby1 + codeblockHeight - 1) >> ycb
	precinctParameters := subband.resolution.precinctParameters

	var codeblocks []*codeblock
	var precincts []*precinctStruct

	for j := cby0; j < cby1; j++ {
		for i := cbx0; i < cbx1; i++ {
			var codeblock codeblock
			codeblock.cbx = i
			codeblock.cby = j
			codeblock.tbx0 = codeblockWidth * i
			codeblock.tby0 = codeblockHeight * j
			codeblock.tbx1 = codeblockWidth * (i + 1)
			codeblock.tby1 = codeblockHeight * (j + 1)

			codeblock.tbx0_ = maxuint32(subband.tbx0, codeblock.tbx0)
			codeblock.tby0_ = maxuint32(subband.tby0, codeblock.tby0)
			codeblock.tbx1_ = minuint32(subband.tbx1, codeblock.tbx1)
			codeblock.tby1_ = minuint32(subband.tby1, codeblock.tby1)

			// Calculate precinct number for this codeblock, codeblock position
			// should be relative to its subband, use actual dimension and position
			// See comment about codeblock group width and height
			pi := uint32(math.Floor(float64(codeblock.tbx0_-subband.tbx0) / float64(precinctParameters.precinctWidthInSubband)))
			pj := uint32(math.Floor(float64(codeblock.tby0_-subband.tby0) / float64(precinctParameters.precinctHeightInSubband)))

			codeblock.precinctNumber = pi + pj*precinctParameters.numPrecinctsWide
			codeblock.subbandType = subband.subbandType
			codeblock.Lblock = 3

			if codeblock.tbx1_ <= codeblock.tbx0_ || codeblock.tby1_ <= codeblock.tby0_ {
				continue
			}
			codeblocks = append(codeblocks, &codeblock)

			// building precinct for the sub-band
			var precinct *precinctStruct

			if codeblock.precinctNumber < uint32(len(precincts)) {
				precinct = precincts[codeblock.precinctNumber]

				if i < precinct.cbxMin {
					precinct.cbxMin = i
				} else if i > precinct.cbxMax {
					precinct.cbxMax = i
				}
				if j < precinct.cbyMin {
					precinct.cbxMin = j
				} else if j > precinct.cbyMax {
					precinct.cbyMax = j
				}
			} else {
				precinct = new(precinctStruct)
				precinct.cbxMin = i
				precinct.cbyMin = j
				precinct.cbxMax = i
				precinct.cbyMax = j

				precincts = append(precincts, precinct)
			}

			codeblock.precinct = precinct
		}
	}

	var codeblockParameters codeblockParameters
	codeblockParameters.codeblockWidth = xcb
	codeblockParameters.codeblockHeight = ycb
	codeblockParameters.numcodeblockwide = cbx1 - cbx0 + 1
	codeblockParameters.numcodeblockhigh = cby1 - cby0 + 1

	subband.codeblockParameters = &codeblockParameters
	subband.codeblocks = codeblocks
	subband.precincts = precincts
}

func (header *Header) buildPackets() {
	siz := header.Size
	tileIndex := header.currentTile.index
	tile := header.tiles[tileIndex]
	componentsCount := int(siz.Csiz)

	// Creating resolutions and sub-bands for each component
	for c := 0; c < componentsCount; c++ {
		component := tile.components[c]
		decompositionLevelsCount := component.codingStyleParameters.NumberOfLevels

		// Section B.5 Resolution levels and sub-bands
		var resolutions []*resolution
		var subbands []*subbandStruct

		fmt.Printf("decompositionLevelsCount = %d\n", decompositionLevelsCount)

		for r := uint8(0); r <= decompositionLevelsCount; r++ {
			blocksDimensions := header.getBlocksDimensions(component, r)

			fmt.Printf("BlockDimensions = %v\n", blocksDimensions)
			var resolution resolution

			scale := 1 << (decompositionLevelsCount - r)
			resolution.trx0 = uint32(math.Ceil(float64(component.tcx0) / float64(scale)))
			resolution.try0 = uint32(math.Ceil(float64(component.tcy0) / float64(scale)))
			resolution.trx1 = uint32(math.Ceil(float64(component.tcx1) / float64(scale)))
			resolution.try1 = uint32(math.Ceil(float64(component.tcy1) / float64(scale)))
			resolution.resLevel = r

			resolution.buildPrecincts(header, blocksDimensions)
			resolutions = append(resolutions, &resolution)

			var subband *subbandStruct
			if r == 0 {
				// one sub-band (LL) with last decomposition

				subband = new(subbandStruct)
				subband.subbandType = "LL"
				subband.tbx0 = uint32(math.Ceil(float64(component.tcx0) / float64(scale)))
				subband.tby0 = uint32(math.Ceil(float64(component.tcy0) / float64(scale)))
				subband.tbx1 = uint32(math.Ceil(float64(component.tcx1) / float64(scale)))
				subband.tby1 = uint32(math.Ceil(float64(component.tcy1) / float64(scale)))
				subband.resolution = &resolution
				subband.buildCodeblocks(header, blocksDimensions)

				subbands = append(subbands, subband)
				resolution.subbands = append(resolution.subbands, subband)
			} else {
				var bscale = 1 << (decompositionLevelsCount - r + 1)
				// three sub-bands (HL, LH and HH) with rest of decompositions

				subband = new(subbandStruct)
				subband.subbandType = "HL"
				subband.tbx0 = uint32(math.Ceil(float64(component.tcx0)/float64(bscale) - 0.5))
				subband.tby0 = uint32(math.Ceil(float64(component.tcy0) / float64(bscale)))
				subband.tbx1 = uint32(math.Ceil(float64(component.tcx1)/float64(bscale) - 0.5))
				subband.tby1 = uint32(math.Ceil(float64(component.tcy1) / float64(bscale)))
				subband.resolution = &resolution
				subband.buildCodeblocks(header, blocksDimensions)
				subbands = append(subbands, subband)
				resolution.subbands = append(resolution.subbands, subband)

				subband = new(subbandStruct)
				subband.subbandType = "LH"
				subband.tbx0 = uint32(math.Ceil(float64(component.tcx0) / float64(bscale)))
				subband.tby0 = uint32(math.Ceil(float64(component.tcy0)/float64(bscale) - 0.5))
				subband.tbx1 = uint32(math.Ceil(float64(component.tcx1) / float64(bscale)))
				subband.tby1 = uint32(math.Ceil(float64(component.tcy1)/float64(bscale) - 0.5))
				subband.resolution = &resolution
				subband.buildCodeblocks(header, blocksDimensions)
				subbands = append(subbands, subband)
				resolution.subbands = append(resolution.subbands, subband)

				subband = new(subbandStruct)
				subband.subbandType = "HH"
				subband.tbx0 = uint32(math.Ceil(float64(component.tcx0)/float64(bscale) - 0.5))
				subband.tby0 = uint32(math.Ceil(float64(component.tcy0)/float64(bscale) - 0.5))
				subband.tbx1 = uint32(math.Ceil(float64(component.tcx1)/float64(bscale) - 0.5))
				subband.tby1 = uint32(math.Ceil(float64(component.tcy1)/float64(bscale) - 0.5))
				subband.resolution = &resolution
				subband.buildCodeblocks(header, blocksDimensions)
				subbands = append(subbands, subband)
				resolution.subbands = append(resolution.subbands, subband)
			}

			fmt.Println(resolution)
			fmt.Println(resolution.precinctParameters)
			fmt.Println(resolution.subbands[0])
			fmt.Println(resolution.subbands[0].codeblocks[0])
			fmt.Println(resolution.subbands[0].precincts[0])
		}

		component.resolutions = resolutions
		component.subbands = subbands
	}

	switch tile.codingStyleDefaultParameters.ProgressionOrder {
	case 0:
		tile.packetsIterator = newLayerResolutionComponentPositionIterator(header)
	default:
		log.Printf("Unsupported ProgressionOrder = %d\n", tile.codingStyleDefaultParameters.ProgressionOrder)
	}
	// Generate the packets sequence
	/*		var progressionOrder = tile.codingStyleDefaultParameters.progressionOrder;
			switch (progressionOrder) {
			  case 0:
				tile.packetsIterator = new LayerResolutionComponentPositionIterator(
				  context
				);
				break;
			  case 1:
				tile.packetsIterator = new ResolutionLayerComponentPositionIterator(
				  context
				);
				break;
			  case 2:
				tile.packetsIterator = new ResolutionPositionComponentLayerIterator(
				  context
				);
				break;
			  case 3:
				tile.packetsIterator = new PositionComponentResolutionLayerIterator(
				  context
				);
				break;
			  case 4:
				tile.packetsIterator = new ComponentPositionResolutionLayerIterator(
				  context
				);
				break;
			  default:
				throw new JpxError(`Unsupported progression order ${progressionOrder}`);
			}*/
}

func (header *Header) GetImageWidth() uint32 {
	return header.Size.Xsiz - header.Size.XOsiz
}

func (header *Header) GetImageHeight() uint32 {
	return header.Size.Ysiz - header.Size.YOsiz
}

// GetImageULX returns the horizontal upper-left coordinate of the image in the reference grid.
func (header *Header) GetImageULX() uint32 {
	return header.Size.XOsiz
}

// GetImageULY returns the vertical upper-left coordinate of the image in the reference grid.
func (header *Header) GetImageULY() uint32 {
	return header.Size.YOsiz
}

// GetNomTileWidth returns the nominal width of the tiles in the reference grid.
func (header *Header) GetNomTileWidth() uint32 {
	return header.Size.XTsiz
}

// GetNomTileHeight returns the nominal height of the tiles in the reference grid.
func (header *Header) GetNomTileHeight() uint32 {
	return header.Size.YTsiz
}

func (header *Header) GetTilingOrigin() (uint32, uint32) {
	return header.Size.XTOsiz, header.Size.YTOsiz
}

func (header *Header) IsComponentSigned(component int) bool {
	return header.Components[component].Signed
}

func (header *Header) GetComponentBitDepth(component int) uint8 {
	return header.Components[component].BitDepth
}

func (header *Header) GetNumComponents() uint16 {
	return header.Size.Csiz
}

// GetComponentSubsX returns the component sub-sampling factor, with respect to the reference grid, along the horizontal direction for the given component.
func (header *Header) GetComponentSubsX(component int) uint8 {
	return header.Components[component].XRsiz
}

// GetComponentSubsY returns the component sub-sampling factor, with respect to the reference grid, along the vertical direction for the given component.
func (header *Header) GetComponentSubsY(component int) uint8 {
	return header.Components[component].YRsiz
}

type size struct {
	// Lsiz is the length of Siz marker segment in bytes (not including the marker). Lcod = 38 + 3 ⋅ Csiz
	Lsiz uint16
	// Rsiz denotes the capabilities that a decoder must implement to successfully decode the codestream.
	Rsiz uint16
	// Xsiz denotes the width of the reference grid.
	Xsiz uint32
	// Ysiz denotes the height of the reference grid.
	Ysiz uint32
	// XOsiz denotes the horizontal offset from the origin of the reference grid to the left side of the image area.
	XOsiz uint32
	// YOsiz denotes the vertical offset from the origin of the reference grid to the top side of the image area.
	YOsiz uint32
	// XTsiz denotes the width of one reference tile with respect to the reference grid.
	XTsiz uint32
	// YTsiz denotes the height of one reference tile with respect to the reference grid.
	YTsiz uint32
	// XTOsiz denotes the horizontal offset from the origin of the reference grid to the left side of the first tile.
	XTOsiz uint32
	// YTOsiz denotes the vertical offset from the origin of the reference grid to the top side of the first tile.
	YTOsiz uint32
	// Csiz denotes the number of components in the image
	Csiz uint16
}

type componentSize struct {
	// BitDepth denotes the precision of the current component.
	BitDepth uint8
	// Signed denotes whether the current component is stored using a signed datatype.
	Signed bool
	// XRsiz denotes the horizontal separation of a sample of the current component with respect to the reference grid.
	XRsiz uint8
	// YRsiz denotes the vertical separation of a sample of the current component with respect to the reference grid.
	YRsiz uint8

	x0     uint32
	x1     uint32
	y0     uint32
	y1     uint32
	width  uint32
	height uint32
}

func (component *componentSize) calculateComponentDimensions(siz *size) {
	// Section B.2 Component mapping
	component.x0 = uint32(math.Ceil(float64(siz.XOsiz) / float64(component.XRsiz)))
	component.x1 = uint32(math.Ceil(float64(siz.Xsiz) / float64(component.XRsiz)))
	component.y0 = uint32(math.Ceil(float64(siz.YOsiz) / float64(component.YRsiz)))
	component.y1 = uint32(math.Ceil(float64(siz.Ysiz) / float64(component.YRsiz)))
	component.width = component.x1 - component.x0
	component.height = component.y1 - component.y0
}

type CodingStyle struct {
	EntropyCoder bool
	SOPMarker    bool
	EPHMarker    bool

	ProgressionOrder                ProgressionOrder
	NumberOfLayers                  uint16
	MultipleComponentTransformation bool

	NumberOfLevels uint8
	Xcb            uint8
	Ycb            uint8

	ArithmeticCodingBypass   bool
	ResetContextProbabilties bool
	TerminateOnCodingPass    bool
	VerticallyCasualContext  bool
	PredictableTermination   bool
	SegmentationSymbols      bool

	ReversableFilter bool
}

type ProgressionOrder uint8

const (
	LayerResolutionLevelComponentPosition ProgressionOrder = 0
	ResolutionLevelLayerComponentPosition ProgressionOrder = 1
	ResolutionLevelPositionComponentLayer ProgressionOrder = 2
	PositionComponentResolutionLevelLayer ProgressionOrder = 3
	ComponentPositionResolutionLevelLayer ProgressionOrder = 4
)

var progressionOrderMap = map[uint8]ProgressionOrder{
	0: LayerResolutionLevelComponentPosition,
	1: ResolutionLevelLayerComponentPosition,
	2: ResolutionLevelPositionComponentLayer,
	3: PositionComponentResolutionLevelLayer,
	4: ComponentPositionResolutionLevelLayer,
}

type Quantization struct {
	QuantizationStyle uint8
	GuardBits         uint8

	SPqcd []QuantizationStepSize
}

type QuantizationStepSize struct {
	Mantissa uint16
	Exponent uint16
}

func Decode(r io.Reader) error {
	var header Header
	var buf []byte
	var err error
	var n int
	done := false

	buf = make([]byte, 4028)

	n, err = r.Read(buf[:2])
	if err != nil {
		return err
	}

	for n > 0 {
		// Process data
		if buf[0] == Marker {
			switch buf[1] {
			case EOC:
				done = true
			case SOC:
			case SIZ:
				type cSize struct {
					Ssiz  uint8
					XRsiz uint8
					YRsiz uint8
				}

				err = binary.Read(r, binary.BigEndian, &header.Size)
				if err != nil {
					return err
				}

				fmt.Println(header.Size)

				var components []cSize
				components = make([]cSize, header.Size.Csiz)

				err = binary.Read(r, binary.BigEndian, &components)
				if err != nil {
					return err
				}

				header.Components = make([]componentSize, header.Size.Csiz)

				for i := range components {
					header.Components[i].BitDepth = (components[i].Ssiz & 0x7F) + 1
					header.Components[i].Signed = (components[i].Ssiz >> 7) == 1
					header.Components[i].XRsiz = components[i].XRsiz
					header.Components[i].YRsiz = components[i].YRsiz

					header.Components[i].calculateComponentDimensions(&header.Size)
				}

				header.calculateTileGrids()
			case COD:
				type codStruct struct {
					Lcod uint16
					Scod uint8
					//SGcod uint32
					ProgressionOrder        uint8
					NumberOfLayers          uint16
					ComponentTransformation uint8

					NumberOfLevels        uint8
					CodeBlockWidth        uint8
					CodeBlockHeight       uint8
					CodeBlockStyle        uint8
					WaveletTransformation uint8
				}

				var codS codStruct
				err = binary.Read(r, binary.BigEndian, &codS)
				if err != nil {
					return err
				}

				header.COD.EntropyCoder = (codS.Scod & 0x1) == 1
				header.COD.SOPMarker = ((codS.Scod >> 1) & 0x1) == 1
				header.COD.EPHMarker = ((codS.Scod >> 2) & 0x1) == 1

				header.COD.ProgressionOrder = progressionOrderMap[codS.ProgressionOrder]
				header.COD.NumberOfLayers = codS.NumberOfLayers
				header.COD.MultipleComponentTransformation = codS.ComponentTransformation == 1

				header.COD.NumberOfLevels = codS.NumberOfLevels
				header.COD.Xcb = codS.CodeBlockWidth + 2
				header.COD.Ycb = codS.CodeBlockHeight + 2

				header.COD.ArithmeticCodingBypass = (codS.CodeBlockStyle & 0x1) == 1
				header.COD.ResetContextProbabilties = ((codS.CodeBlockStyle >> 1) & 0x1) == 1
				header.COD.TerminateOnCodingPass = ((codS.CodeBlockStyle >> 2) & 0x1) == 1
				header.COD.VerticallyCasualContext = ((codS.CodeBlockStyle >> 3) & 0x1) == 1
				header.COD.PredictableTermination = ((codS.CodeBlockStyle >> 4) & 0x1) == 1
				header.COD.SegmentationSymbols = ((codS.CodeBlockStyle >> 5) & 0x1) == 1

				header.COD.ReversableFilter = codS.WaveletTransformation == 1

				//fmt.Println(cod)
				// TODO: Missing Precinct size
			case QCD:
				type qcdStruct struct {
					Lqcd uint16
					Sqcd uint8
				}

				var qcd qcdStruct
				err = binary.Read(r, binary.BigEndian, &qcd)
				if err != nil {
					return err
				}

				header.QCD.QuantizationStyle = qcd.Sqcd & 0x1F
				header.QCD.GuardBits = qcd.Sqcd >> 5

				if header.QCD.QuantizationStyle > 0 {
					// Size of SPqcd is uint16
					var SPqcd []uint16
					SPqcd = make([]uint16, header.COD.NumberOfLevels)

					err = binary.Read(r, binary.BigEndian, &SPqcd)
					if err != nil {
						return err
					}

					header.QCD.SPqcd = make([]QuantizationStepSize, header.COD.NumberOfLevels)

					for i := range SPqcd {
						header.QCD.SPqcd[i].Mantissa = SPqcd[i] & 0x7FF
						header.QCD.SPqcd[i].Exponent = SPqcd[i] >> 11
					}
				} else {
					// Size of SPqcd is uint8
					log.Printf("Size of SPqcd is uint8 - not handling this")
				}
			case COM:
				type comment struct {
					Lcom uint16
					Rcom uint16
				}

				var com comment
				err = binary.Read(r, binary.BigEndian, &com)
				if err != nil {
					return err
				}

				var text []byte
				text = make([]byte, com.Lcom)
				err = binary.Read(r, binary.BigEndian, &text)
				if err != nil {
					return err
				}

				var commentString string

				if com.Rcom == 0 {
					commentString = string(text)
				} else {
					decodedBytes, _ := ioutil.ReadAll(charmap.ISO8859_15.NewDecoder().Reader(bytes.NewReader(text)))
					commentString = string(decodedBytes)
				}

				fmt.Println(commentString)
			case SOT:
				type sotStruct struct {
					Lsot  uint16
					Isot  uint16
					Psot  uint32
					TPsot uint8
					TNsot uint8
				}

				var sot sotStruct
				err = binary.Read(r, binary.BigEndian, &sot)
				if err != nil {
					return err
				}

				var tHeader tileHeader
				tHeader.index = sot.Isot
				tHeader.length = sot.Psot
				tHeader.dataSize = sot.Psot - uint32(sot.Lsot)
				tHeader.partIndex = sot.TPsot
				tHeader.partsCount = sot.TNsot

				if tHeader.partIndex == 0 {
					tHeader.COD = &header.COD
					tHeader.QCD = &header.QCD

					log.Println("JPEG2000: Not handling COC or QCC")
				}

				header.currentTile = &tHeader

				/*dataSize := sot.Psot - uint32(sot.Lsot)
				var data []byte
				data = make([]byte, dataSize)

				err = binary.Read(r, binary.BigEndian, &data)
				if err != nil {
					return err
				}

				// Strip off the start markers
				log.Println("JPEG2000: Assuming next marker is SOD and all defaults are used")
				data = data[2:]

				fmt.Println(hex.EncodeToString(data))*/

				// If Layer-resolution level-component-position progression
				//for each l = 0,..., L-1
				//	for each r = 0,..., Nmax
				//		for each i = 0,..., Csiz-1
				//			for each k = 0,..., numprecincts-1
				//				packet for component i, resolution level r, layer l, and precinct k.

				//done = true
			case SOD:
				if header.currentTile.partIndex == 0 {
					header.initialiseTile()
					header.buildPackets()
				}

				var data []byte
				data = make([]byte, header.currentTile.dataSize-4)
				// TODO: is -4 correct? had -2 for the marker before, but when this is set, there is the EOC marker included

				err = binary.Read(r, binary.BigEndian, &data)
				if err != nil {
					log.Println(err)
					return err
				}

				fmt.Println(hex.EncodeToString(data))
				header.parseTilePackets(data)
			default:
				fmt.Printf("Found marker: %s\n", hex.EncodeToString(buf[:2]))
			}

		}
		if !done {
			n, err = r.Read(buf[:2])
			if err != nil {
				return err
			}
		} else {
			break
		}
	}

	tiles := header.transformComponents()
	//width := header.Size.Xsiz - header.Size.XOsiz
	//height := header.Size.Ysiz - header.Size.YOsiz
	//componentsCount := header.Size.Csiz

	fmt.Println(tiles)

	fmt.Println(header)
	fmt.Println(header.tiles[0])
	fmt.Println(header.tiles[0].components[0])

	return nil
}
