package gofont

import "errors"

// Glyph flag constants for simple glyphs
const (
	glyphFlagOnCurve   uint8 = 0x01
	glyphFlagXShort    uint8 = 0x02
	glyphFlagYShort    uint8 = 0x04
	glyphFlagRepeat    uint8 = 0x08
	glyphFlagXSame     uint8 = 0x10 // positive if XShort, same as previous if !XShort
	glyphFlagYSame     uint8 = 0x20 // positive if YShort, same as previous if !YShort
)

// Composite glyph flag constants
const (
	compositeArg1And2AreWords   uint16 = 0x0001
	compositeArgsAreXYValues    uint16 = 0x0002
	compositeRoundXYToGrid      uint16 = 0x0004
	compositeWeHaveAScale       uint16 = 0x0008
	compositeMoreComponents     uint16 = 0x0020
	compositeWeHaveAnXAndYScale uint16 = 0x0040
	compositeWeHaveATwoByTwo    uint16 = 0x0080
)

type GlyphHeader struct {
	numberOfContours int16
	xMin             int16
	yMin             int16
	xMax             int16
	yMax             int16
}

type SimpleGlyph struct {
	endPtsOfContours []uint16
	instructions     []byte
	flags            []uint8
	xCoordinates     []int16
	yCoordinates     []int16
}

type GlyphComponent struct {
	flags      uint16
	glyphIndex uint16
	arg1       int16
	arg2       int16
	transform  [4]int16 // optional 2x2 matrix (scale01, scale10, scale00, scale11)
	hasScale   bool
	hasXYScale bool
	has2x2     bool
}

type CompositeGlyph struct {
	components []GlyphComponent
}

type Glyph struct {
	header         GlyphHeader
	simpleGlyph    *SimpleGlyph
	compositeGlyph *CompositeGlyph
}

func parseGlyf(glyfData []byte, loca *Loca) ([]*Glyph, error) {
	numGlyphs := len(loca.offsets) - 1
	glyphs := make([]*Glyph, numGlyphs)

	for i := 0; i < numGlyphs; i++ {
		start := loca.offsets[i]
		end := loca.offsets[i+1]
		size := end - start

		if size == 0 {
			// Empty glyph (e.g., .notdef or space)
			glyphs[i] = &Glyph{}
			continue
		}

		if int(end) > len(glyfData) {
			return nil, errors.New("glyph offset out of bounds")
		}

		glyphData := glyfData[start:end]
		glyph, err := parseGlyph(glyphData)
		if err != nil {
			return nil, err
		}
		glyphs[i] = glyph
	}

	return glyphs, nil
}

func parseGlyph(data []byte) (*Glyph, error) {
	if len(data) < 10 {
		return nil, errors.New("glyph data too small")
	}

	binary := BinaryFrom(data, false)
	glyph := &Glyph{}

	glyph.header.numberOfContours = binary.I16()
	glyph.header.xMin = binary.I16()
	glyph.header.yMin = binary.I16()
	glyph.header.xMax = binary.I16()
	glyph.header.yMax = binary.I16()

	if glyph.header.numberOfContours >= 0 {
		sg, err := parseSimpleGlyph(binary, int(glyph.header.numberOfContours))
		if err != nil {
			return nil, err
		}
		glyph.simpleGlyph = sg
	} else {
		cg, err := parseCompositeGlyph(binary)
		if err != nil {
			return nil, err
		}
		glyph.compositeGlyph = cg
	}

	return glyph, nil
}

func parseSimpleGlyph(binary Binary, numContours int) (*SimpleGlyph, error) {
	sg := &SimpleGlyph{}

	if numContours == 0 {
		return sg, nil
	}

	sg.endPtsOfContours = make([]uint16, numContours)
	for i := 0; i < numContours; i++ {
		sg.endPtsOfContours[i] = binary.U16()
	}

	numPoints := int(sg.endPtsOfContours[numContours-1]) + 1

	instructionLength := binary.U16()
	sg.instructions = binary.Read(int(instructionLength))

	// Read flags (run-length encoded)
	sg.flags = make([]uint8, numPoints)
	for i := 0; i < numPoints; {
		flag := binary.U8()
		sg.flags[i] = flag
		i++
		if flag&glyphFlagRepeat != 0 {
			repeatCount := int(binary.U8())
			for j := 0; j < repeatCount && i < numPoints; j++ {
				sg.flags[i] = flag
				i++
			}
		}
	}

	// Read X coordinates (delta-encoded)
	sg.xCoordinates = make([]int16, numPoints)
	var x int16
	for i := 0; i < numPoints; i++ {
		flag := sg.flags[i]
		if flag&glyphFlagXShort != 0 {
			dx := int16(binary.U8())
			if flag&glyphFlagXSame == 0 {
				dx = -dx
			}
			x += dx
		} else if flag&glyphFlagXSame != 0 {
			// x same as previous (delta = 0)
		} else {
			x += binary.I16()
		}
		sg.xCoordinates[i] = x
	}

	// Read Y coordinates (delta-encoded)
	sg.yCoordinates = make([]int16, numPoints)
	var y int16
	for i := 0; i < numPoints; i++ {
		flag := sg.flags[i]
		if flag&glyphFlagYShort != 0 {
			dy := int16(binary.U8())
			if flag&glyphFlagYSame == 0 {
				dy = -dy
			}
			y += dy
		} else if flag&glyphFlagYSame != 0 {
			// y same as previous (delta = 0)
		} else {
			y += binary.I16()
		}
		sg.yCoordinates[i] = y
	}

	return sg, nil
}

func parseCompositeGlyph(binary Binary) (*CompositeGlyph, error) {
	cg := &CompositeGlyph{}

	for {
		comp := GlyphComponent{}
		comp.flags = binary.U16()
		comp.glyphIndex = binary.U16()

		if comp.flags&compositeArg1And2AreWords != 0 {
			comp.arg1 = binary.I16()
			comp.arg2 = binary.I16()
		} else {
			comp.arg1 = int16(binary.U8())
			comp.arg2 = int16(binary.U8())
		}

		if comp.flags&compositeWeHaveAScale != 0 {
			comp.hasScale = true
			comp.transform[0] = binary.I16() // F2Dot14
		} else if comp.flags&compositeWeHaveAnXAndYScale != 0 {
			comp.hasXYScale = true
			comp.transform[0] = binary.I16()
			comp.transform[3] = binary.I16()
		} else if comp.flags&compositeWeHaveATwoByTwo != 0 {
			comp.has2x2 = true
			comp.transform[0] = binary.I16()
			comp.transform[1] = binary.I16()
			comp.transform[2] = binary.I16()
			comp.transform[3] = binary.I16()
		}

		cg.components = append(cg.components, comp)

		if comp.flags&compositeMoreComponents == 0 {
			break
		}
	}

	return cg, nil
}

func writeGlyf(glyphs []*Glyph) ([]byte, []uint32) {
	// Write each glyph and build loca offsets simultaneously
	locaOffsets := make([]uint32, len(glyphs)+1)
	var buf []byte
	var offset uint32

	for i, g := range glyphs {
		locaOffsets[i] = offset
		sz := glyphEncodedSize(g)
		if sz > 0 {
			start := len(buf)
			// Pad to even boundary (short loca format requires even offsets)
			paddedSize := sz
			if paddedSize%2 != 0 {
				paddedSize++
			}
			buf = append(buf, make([]byte, paddedSize)...)
			binary := BinaryFrom(buf[start:], false)
			writeGlyph(binary, g)
			offset += uint32(paddedSize)
		}
	}
	locaOffsets[len(glyphs)] = offset

	return buf, locaOffsets
}

// glyphSize returns the actual encoded byte size of a glyph.
// It writes to a temporary buffer to get exact size.
func glyphEncodedSize(g *Glyph) int {
	if g == nil || (g.header.numberOfContours == 0 && g.simpleGlyph == nil && g.compositeGlyph == nil) {
		return 0
	}

	// Estimate generously, write to temp buffer, return actual size
	est := 10 + 1000 // generous estimate
	tmp := make([]byte, est)
	binary := BinaryFrom(tmp, false)
	writeGlyph(binary, g)
	return binary.Offset()
}

func writeGlyph(binary Binary, g *Glyph) {
	binary.PutU16(uint16(g.header.numberOfContours))
	binary.PutU16(uint16(g.header.xMin))
	binary.PutU16(uint16(g.header.yMin))
	binary.PutU16(uint16(g.header.xMax))
	binary.PutU16(uint16(g.header.yMax))

	if g.simpleGlyph != nil {
		writeSimpleGlyph(binary, g.simpleGlyph)
	} else if g.compositeGlyph != nil {
		writeCompositeGlyph(binary, g.compositeGlyph)
	}
}

func writeSimpleGlyph(binary Binary, sg *SimpleGlyph) {
	for _, ep := range sg.endPtsOfContours {
		binary.PutU16(ep)
	}

	binary.PutU16(uint16(len(sg.instructions)))
	binary.Append(sg.instructions)

	// Recompute flags from coordinate deltas
	numPoints := len(sg.xCoordinates)
	flags := make([]uint8, numPoints)
	var prevX, prevY int16
	for i := 0; i < numPoints; i++ {
		dx := sg.xCoordinates[i] - prevX
		dy := sg.yCoordinates[i] - prevY
		prevX = sg.xCoordinates[i]
		prevY = sg.yCoordinates[i]

		f := sg.flags[i] & glyphFlagOnCurve // preserve on-curve bit

		if dx == 0 {
			f |= glyphFlagXSame
		} else if dx > 0 && dx <= 255 || dx < 0 && dx >= -255 {
			f |= glyphFlagXShort
			if dx > 0 {
				f |= glyphFlagXSame // positive short
			}
		}
		// else: 2-byte delta, XShort=0, XSame=0

		if dy == 0 {
			f |= glyphFlagYSame
		} else if dy > 0 && dy <= 255 || dy < 0 && dy >= -255 {
			f |= glyphFlagYShort
			if dy > 0 {
				f |= glyphFlagYSame // positive short
			}
		}
		// else: 2-byte delta, YShort=0, YSame=0

		flags[i] = f
	}

	// Write flags with RLE encoding
	for i := 0; i < numPoints; {
		f := flags[i]
		// Count consecutive identical flags
		repeat := 1
		for i+repeat < numPoints && flags[i+repeat] == f {
			repeat++
		}

		if repeat >= 2 {
			binary.PutU8(f | glyphFlagRepeat)
			binary.PutU8(uint8(repeat - 1))
			i += repeat
		} else {
			binary.PutU8(f)
			i++
		}
	}

	// Write X coordinates (delta-encoded)
	prevX = 0
	for i := 0; i < numPoints; i++ {
		dx := sg.xCoordinates[i] - prevX
		prevX = sg.xCoordinates[i]

		if flags[i]&glyphFlagXShort != 0 {
			if dx >= 0 {
				binary.PutU8(uint8(dx))
			} else {
				binary.PutU8(uint8(-dx))
			}
		} else if flags[i]&glyphFlagXSame == 0 {
			binary.PutU16(uint16(dx))
		}
	}

	// Write Y coordinates (delta-encoded)
	prevY = 0
	for i := 0; i < numPoints; i++ {
		dy := sg.yCoordinates[i] - prevY
		prevY = sg.yCoordinates[i]

		if flags[i]&glyphFlagYShort != 0 {
			if dy >= 0 {
				binary.PutU8(uint8(dy))
			} else {
				binary.PutU8(uint8(-dy))
			}
		} else if flags[i]&glyphFlagYSame == 0 {
			binary.PutU16(uint16(dy))
		}
	}
}

func writeCompositeGlyph(binary Binary, cg *CompositeGlyph) {
	for _, comp := range cg.components {
		binary.PutU16(comp.flags)
		binary.PutU16(comp.glyphIndex)

		if comp.flags&compositeArg1And2AreWords != 0 {
			binary.PutU16(uint16(comp.arg1))
			binary.PutU16(uint16(comp.arg2))
		} else {
			binary.PutU8(uint8(comp.arg1))
			binary.PutU8(uint8(comp.arg2))
		}

		if comp.hasScale {
			binary.PutU16(uint16(comp.transform[0]))
		} else if comp.hasXYScale {
			binary.PutU16(uint16(comp.transform[0]))
			binary.PutU16(uint16(comp.transform[3]))
		} else if comp.has2x2 {
			binary.PutU16(uint16(comp.transform[0]))
			binary.PutU16(uint16(comp.transform[1]))
			binary.PutU16(uint16(comp.transform[2]))
			binary.PutU16(uint16(comp.transform[3]))
		}
	}
}
