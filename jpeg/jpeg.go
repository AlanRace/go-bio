package jpeg

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	Marker uint8 = 0xff
	SOI    uint8 = 0xd8
	APP0   uint8 = 0xe0
	// ...
	DQT  uint8 = 0xdb
	SOF0 uint8 = 0xc0
	DHT  uint8 = 0xc4
	SOS  uint8 = 0xda
	EOI  uint8 = 0xd9

	RST0 uint8 = 0xd0
	RST7 uint8 = 0xd7
)

type QuantizationTable8Bit struct {
	QTNumber  uint8
	Precision uint8
	Values    []uint8
}

type JPEGHeader struct {
	dqt QuantizationTable8Bit
	// First dimension is DC (0) or AC (1). Second dimension is  HT number
	huffmanTables [2][4]HuffmanTable
}

type FrameComponent struct {
	ComponentID              uint8
	VerticalSamplingFactor   uint8
	HorizontalSamplingFactor uint8
	QuantizationTableNumber  uint8
}

type StartOfFrame struct {
	Length        uint16
	Precision     uint8
	ImageHeight   uint16
	ImageWidth    uint16
	NumComponents uint8
}

type ScanComponent struct {
	ComponentID uint8
	ACTable     *HuffmanTable
	DCTable     *HuffmanTable
}

type JPEGBody struct {
	sof0            StartOfFrame
	frameComponents []FrameComponent
	scanComponents  []ScanComponent
}

func DecodeHeader(r io.Reader) (*JPEGHeader, error) {
	var header JPEGHeader
	var buf []byte
	var err error
	var n int
	done := false

	buf = make([]byte, 4028)

	n, err = r.Read(buf[:2])
	if err != nil {
		return nil, err
	}

	for n > 0 {
		// Process data
		if buf[0] == Marker {
			switch buf[1] {
			case SOI:
				//fmt.Println("Found SOI")
			case EOI:
				//fmt.Println("Found EOI")
				done = true
			case DQT:
				//fmt.Println("Found DQT")
				var table QuantizationTable8Bit
				type dqt struct {
					Length      uint16
					Information uint8
				}
				var qt dqt

				err = binary.Read(r, binary.BigEndian, &qt)
				if err != nil {
					return nil, err
				}

				table.QTNumber = qt.Information & 0x0F
				table.Precision = (qt.Information & 0xF0) >> 4
				table.Values = make([]uint8, 64)

				err = binary.Read(r, binary.BigEndian, &table.Values)
				if err != nil {
					return nil, err
				}

				header.dqt = table
			case DHT:
				//fmt.Println("Found DHT")
				type dht struct {
					Length          uint16
					Information     uint8
					NumberOfSymbols [16]uint8
				}

				var ht dht

				err = binary.Read(r, binary.BigEndian, &ht)
				if err != nil {
					return nil, err
				}

				htNumber := ht.Information & 0x0F
				htType := (ht.Information & 16) >> 4

				copy(header.huffmanTables[htType][htNumber].NumberOfSymbols[:], ht.NumberOfSymbols[:])

				for _, num := range ht.NumberOfSymbols {
					header.huffmanTables[htType][htNumber].nCodes += num
				}

				header.huffmanTables[htType][htNumber].Symbols = make([]uint8, header.huffmanTables[htType][htNumber].nCodes)

				err = binary.Read(r, binary.BigEndian, &header.huffmanTables[htType][htNumber].Symbols)
				if err != nil {
					return nil, err
				}

				header.huffmanTables[htType][htNumber].generateLUT()
			default:
				fmt.Printf("Found marker: %s\n", hex.EncodeToString(buf[:2]))
			}

		}

		if !done {
			n, err = r.Read(buf[:2])
			if err != nil {
				return &header, err
			}
		} else {
			break
		}
	}

	return &header, nil
}

type Data struct {
	Width    int
	Height   int
	Channels int
	Data     []uint8
}

// DecodeBody
func (header *JPEGHeader) DecodeBody(r io.Reader) (*Data, error) {
	var body JPEGBody
	var buf []byte
	var err error
	var n int

	buf = make([]byte, 256)

	n, err = r.Read(buf[:2])
	if err != nil {
		return nil, err
	}

	done := false

	for n > 0 {
		// Process data
		if buf[0] == Marker {
			switch buf[1] {
			case SOI:
				//fmt.Println("Found SOI")
			case EOI:
				//fmt.Println("Found EOI")
			case SOF0:
				//fmt.Println("Found SOF0")
				err = binary.Read(r, binary.BigEndian, &body.sof0)
				if err != nil {
					return nil, err
				}

				body.frameComponents = make([]FrameComponent, body.sof0.NumComponents)

				n, err = r.Read(buf[:(body.sof0.NumComponents * 3)])
				if err != nil {
					return nil, err
				}

				for index := range body.frameComponents {
					offset := index * 3

					body.frameComponents[index].ComponentID = buf[offset]
					body.frameComponents[index].VerticalSamplingFactor = buf[offset+1] & 0x0F
					body.frameComponents[index].HorizontalSamplingFactor = (buf[offset+1] & 0xF0) >> 4
					body.frameComponents[index].QuantizationTableNumber = buf[offset+2]
				}
			case SOS:
				//fmt.Println("Found SOS")
				n, err = r.Read(buf[:3])
				if err != nil {
					return nil, err
				}

				body.scanComponents = make([]ScanComponent, buf[2])

				n, err = r.Read(buf[:(buf[2]*2)+3])
				if err != nil {
					return nil, err
				}

				for index := range body.scanComponents {
					offset := index * 2

					body.scanComponents[index].ComponentID = buf[offset]
					acTableIndex := buf[offset+1] & 0x0F
					dcTableIndex := (buf[offset+1] & 0xF0) >> 4

					body.scanComponents[index].ACTable = &header.huffmanTables[1][acTableIndex]
					body.scanComponents[index].DCTable = &header.huffmanTables[0][dcTableIndex]
				}

				return decodeSOS(header, &body, r)
			default:
				fmt.Printf("Found marker: %s\n", hex.EncodeToString(buf[:2]))
			}
		} else {
			if !done {
				fmt.Printf("Found unknown data: %s\n", hex.EncodeToString(buf[:2]))
				done = true
			}
		}

		n, err = r.Read(buf[:2])
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
