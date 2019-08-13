package jpeg2000

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/text/encoding/charmap"
)

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

type Header struct {
	Size        size
	Components  []componentSize
	CodingStyle CodingStyleDefault
	QCD         QuantizationDefault
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
}

type CodingStyleDefault struct {
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

type QuantizationDefault struct {
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
				}
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

				header.CodingStyle.EntropyCoder = (codS.Scod & 0x1) == 1
				header.CodingStyle.SOPMarker = ((codS.Scod >> 1) & 0x1) == 1
				header.CodingStyle.EPHMarker = ((codS.Scod >> 2) & 0x1) == 1

				header.CodingStyle.ProgressionOrder = progressionOrderMap[codS.ProgressionOrder]
				header.CodingStyle.NumberOfLayers = codS.NumberOfLayers
				header.CodingStyle.MultipleComponentTransformation = codS.ComponentTransformation == 1

				header.CodingStyle.NumberOfLevels = codS.NumberOfLevels
				header.CodingStyle.Xcb = codS.CodeBlockWidth + 2
				header.CodingStyle.Ycb = codS.CodeBlockHeight + 2

				header.CodingStyle.ArithmeticCodingBypass = (codS.CodeBlockStyle & 0x1) == 1
				header.CodingStyle.ResetContextProbabilties = ((codS.CodeBlockStyle >> 1) & 0x1) == 1
				header.CodingStyle.TerminateOnCodingPass = ((codS.CodeBlockStyle >> 2) & 0x1) == 1
				header.CodingStyle.VerticallyCasualContext = ((codS.CodeBlockStyle >> 3) & 0x1) == 1
				header.CodingStyle.PredictableTermination = ((codS.CodeBlockStyle >> 4) & 0x1) == 1
				header.CodingStyle.SegmentationSymbols = ((codS.CodeBlockStyle >> 5) & 0x1) == 1

				header.CodingStyle.ReversableFilter = codS.WaveletTransformation == 1

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
					SPqcd = make([]uint16, header.CodingStyle.NumberOfLevels)

					err = binary.Read(r, binary.BigEndian, &SPqcd)
					if err != nil {
						return err
					}

					header.QCD.SPqcd = make([]QuantizationStepSize, header.CodingStyle.NumberOfLevels)

					for i := range SPqcd {
						header.QCD.SPqcd[i].Mantissa = SPqcd[i] & 0x7FF
						header.QCD.SPqcd[i].Exponent = SPqcd[i] >> 11
					}
				} else {
					// Size of SPqcd is uint8
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

				dataSize := sot.Psot - uint32(sot.Lsot)
				var data []byte
				data = make([]byte, dataSize)

				err = binary.Read(r, binary.BigEndian, &data)
				if err != nil {
					return err
				}

				// Strip off the start markers
				data = data[2:]

				fmt.Println(hex.EncodeToString(data))

				// If Layer-resolution level-component-position progression
				//for each l = 0,..., L-1
				//	for each r = 0,..., Nmax
				//		for each i = 0,..., Csiz-1
				//			for each k = 0,..., numprecincts-1
				//				packet for component i, resolution level r, layer l, and precinct k.

				done = true
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

	fmt.Println(header)

	return nil
}
