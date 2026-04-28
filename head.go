package gofont

import "errors"

type MacStyle uint8

const (
	MacStyleBold      MacStyle = 1 << iota // 0x0001
	MacStyleItalic                         // 0x0002
	MacStyleUnderline                      // 0x0004
	MacStyleOutline                        // 0x0008
	MacStyleShadow                         // 0x0010
	MacStyleCondensed                      // 0x0020
	MacStyleExtended                       // 0x0040
)

type Head struct {
	majorVersion       uint16     // Major version number of the font header table — set to 1.
	minorVersion       uint16     // Minor version number of the font header table — set to 0.
	fontRevision       Fixed16_16 // set by font manufacturer
	checksumAdjustment uint32     //
	magicNumber        uint32     // set to 0x5F0F3CF5
	flags              uint16     // bit 0 - y value of 0 specifies baseline
	unitsPerEm         uint16     // range from 64 to 16384
	created            int64      // date
	modified           int64      // date
	xMin               int16      // for all glyph bounding boxes
	yMin               int16      // for all glyph bounding boxes
	xMax               int16      // for all glyph bounding boxes
	yMax               int16      // for all glyph bounding boxes
	macStyle           uint16     // macintosh style
	lowestRecPPEM      uint16     // smallest readable size in pixels
	fontDirectionHint  int16      // (Deprecated) 0 for mixed directional glyphs, 1 for left-to-right glyphs, -1 for right-to-left glyphs
	indexToLocFormat   int16      // 0 for short offsets, 1 for long
	glyphDataFormat    int16      // 0 for current format
}

func parseHead(data []byte) (*Head, error) {
	if len(data) < 54 {
		return nil, errors.New("head table too small")
	}

	binary := BinaryFrom(data, false)

	head := &Head{
		majorVersion:       binary.U16(),
		minorVersion:       binary.U16(),
		fontRevision:       Fixed16_16{binary.I16(), binary.U16()},
		checksumAdjustment: binary.U32(),
		magicNumber:        binary.U32(),
		flags:              binary.U16(),
		unitsPerEm:         binary.U16(),
		created:            binary.I64(),
		modified:           binary.I64(),
		xMin:               binary.I16(),
		yMin:               binary.I16(),
		xMax:               binary.I16(),
		yMax:               binary.I16(),
		macStyle:           binary.U16(),
		lowestRecPPEM:      binary.U16(),
		fontDirectionHint:  binary.I16(),
		indexToLocFormat:   binary.I16(),
		glyphDataFormat:    binary.I16(),
	}

	return head, nil
}

func writeHead(head *Head) []byte {
	data := make([]byte, 54)
	binary := BinaryFrom(data, false)

	binary.PutU16(head.majorVersion)
	binary.PutU16(head.minorVersion)
	binary.PutFixed16_16(head.fontRevision)
	binary.PutU32(head.checksumAdjustment)
	binary.PutU32(head.magicNumber)
	binary.PutU16(head.flags)
	binary.PutU16(head.unitsPerEm)
	binary.PutU64(uint64(head.created))
	binary.PutU64(uint64(head.modified))
	binary.PutU16(uint16(head.xMin))
	binary.PutU16(uint16(head.yMin))
	binary.PutU16(uint16(head.xMax))
	binary.PutU16(uint16(head.yMax))
	binary.PutU16(head.macStyle)
	binary.PutU16(head.lowestRecPPEM)
	binary.PutU16(uint16(head.fontDirectionHint))
	binary.PutU16(uint16(head.indexToLocFormat))
	binary.PutU16(uint16(head.glyphDataFormat))

	return data
}
