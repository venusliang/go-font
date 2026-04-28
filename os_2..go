package gofont

import "errors"

type OS2 struct {
	version                 uint16
	xAvgCharWidth           int16
	usWeightClass           uint16
	usWidthClass            uint16
	fsType                  uint16
	ySubscriptXSize         int16
	ySubscriptYSize         int16
	ySubscriptXOffset       int16
	ySubscriptYOffset       int16
	ySuperscriptXSize       int16
	ySuperscriptYSize       int16
	ySuperscriptXOffset     int16
	ySuperscriptYOffset     int16
	yStrikeoutSize          int16
	yStrikeoutPosition      int16
	sFamilyClass            int16
	panose                  [10]byte
	ulUnicodeRange1         uint32
	ulUnicodeRange2         uint32
	ulUnicodeRange3         uint32
	ulUnicodeRange4         uint32
	achVendID               [4]byte
	fsSelection             uint16
	usFirstCharIndex        uint16
	usLastCharIndex         uint16
	sTypoAscender           int16
	sTypoDescender          int16
	sTypoLineGap            int16
	usWinAscent             uint16
	usWinDescent            uint16
	ulCodePageRange1        uint32
	ulCodePageRange2        uint32
	sxHeight                int16
	sCapHeight              int16
	usDefaultChar           uint16
	usBreakChar             uint16
	usMaxContext            uint16
	usLowerOpticalPointSize uint16
	usUpperOpticalPointSize uint16
}

func parseOS2(data []byte) (os2 *OS2, err error) {
	if len(data) < 78 {
		return nil, errors.New("os/2 table too short")
	}

	binary := BinaryFrom(data, false)
	version := binary.U16()

	if version > 5 {
		return nil, errors.New("invalid OS/2 table version")
	}

	if version == 0 {
		os2 = os2_format_v0(binary)
	}

	if version == 1 {
		if len(data) < 86 {
			return nil, errors.New("os/2 table too short")
		}
		os2 = os2_format_v1(binary)
	}

	if version == 2 || version == 3 || version == 4 {
		if len(data) < 96 {
			return nil, errors.New("os/2 table too short")
		}

		os2 = os2_format_v4(binary)
	}

	if version == 5 {
		if len(data) < 100 {
			return nil, errors.New("os/2 table too short")
		}

		os2 = os2_format_v5(binary)
	}

	os2.version = version
	return
}

func os2_format_v5(b Binary) *OS2 {
	os2 := os2_format_v4(b)

	os2.usLowerOpticalPointSize = b.U16()
	os2.usUpperOpticalPointSize = b.U16()

	return os2
}

func os2_format_v4(b Binary) *OS2 {
	os2 := os2_format_v1(b)

	os2.sxHeight = b.I16()
	os2.sCapHeight = b.I16()
	os2.usDefaultChar = b.U16()
	os2.usBreakChar = b.U16()
	os2.usMaxContext = b.U16()

	return os2
}

func os2_format_v1(b Binary) *OS2 {
	os2 := os2_format_v0(b)

	os2.ulCodePageRange1 = b.U32()
	os2.ulCodePageRange2 = b.U32()

	return os2
}

func os2_format_v0(b Binary) *OS2 {
	return &OS2{
		xAvgCharWidth:       b.I16(),
		usWeightClass:       b.U16(),
		usWidthClass:        b.U16(),
		fsType:              b.U16(),
		ySubscriptXSize:     b.I16(),
		ySubscriptYSize:     b.I16(),
		ySubscriptXOffset:   b.I16(),
		ySubscriptYOffset:   b.I16(),
		ySuperscriptXSize:   b.I16(),
		ySuperscriptYSize:   b.I16(),
		ySuperscriptXOffset: b.I16(),
		ySuperscriptYOffset: b.I16(),
		yStrikeoutSize:      b.I16(),
		yStrikeoutPosition:  b.I16(),
		sFamilyClass:        b.I16(),
		panose:              [10]uint8(b.Read(10)),
		ulUnicodeRange1:     b.U32(),
		ulUnicodeRange2:     b.U32(),
		ulUnicodeRange3:     b.U32(),
		ulUnicodeRange4:     b.U32(),
		achVendID:           [4]uint8(b.Read(4)),
		fsSelection:         b.U16(),
		usFirstCharIndex:    b.U16(),
		usLastCharIndex:     b.U16(),
		sTypoAscender:       b.I16(),
		sTypoDescender:      b.I16(),
		sTypoLineGap:        b.I16(),
		usWinAscent:         b.U16(),
		usWinDescent:        b.U16(),
	}
}

func writeOS2(os2 *OS2) []byte {
	size := 78
	if os2.version > 0 {
		size += 8
	}

	if os2.version >= 2 {
		size += 10
	}

	if os2.version == 5 {
		size += 4
	}

	data := make([]byte, size)
	binary := BinaryFrom(data, false)

	binary.PutU16(os2.version)
	binary.PutU16(uint16(os2.xAvgCharWidth))
	binary.PutU16(os2.usWeightClass)
	binary.PutU16(os2.usWidthClass)
	binary.PutU16(os2.fsType)
	binary.PutU16(uint16(os2.ySubscriptXSize))
	binary.PutU16(uint16(os2.ySubscriptYSize))
	binary.PutU16(uint16(os2.ySubscriptXOffset))
	binary.PutU16(uint16(os2.ySubscriptYOffset))
	binary.PutU16(uint16(os2.ySuperscriptXSize))
	binary.PutU16(uint16(os2.ySuperscriptYSize))
	binary.PutU16(uint16(os2.ySuperscriptXOffset))
	binary.PutU16(uint16(os2.ySuperscriptYOffset))
	binary.PutU16(uint16(os2.yStrikeoutSize))
	binary.PutU16(uint16(os2.yStrikeoutPosition))
	binary.PutU16(uint16(os2.sFamilyClass))
	binary.Append(os2.panose[:])
	binary.PutU32(os2.ulUnicodeRange1)
	binary.PutU32(os2.ulUnicodeRange2)
	binary.PutU32(os2.ulUnicodeRange3)
	binary.PutU32(os2.ulUnicodeRange4)
	binary.Append(os2.achVendID[:])
	binary.PutU16(os2.fsSelection)
	binary.PutU16(os2.usFirstCharIndex)
	binary.PutU16(os2.usLastCharIndex)
	binary.PutU16(uint16(os2.sTypoAscender))
	binary.PutU16(uint16(os2.sTypoDescender))
	binary.PutU16(uint16(os2.sTypoLineGap))
	binary.PutU16(os2.usWinAscent)
	binary.PutU16(os2.usWinDescent)

	if os2.version >= 1 {
		binary.PutU32(os2.ulCodePageRange1)
		binary.PutU32(os2.ulCodePageRange2)
	}

	if os2.version >= 2 {
		binary.PutU16(uint16(os2.sxHeight))
		binary.PutU16(uint16(os2.sCapHeight))
		binary.PutU16(os2.usDefaultChar)
		binary.PutU16(os2.usBreakChar)
		binary.PutU16(os2.usMaxContext)
	}

	if os2.version == 5 {
		binary.PutU16(os2.usLowerOpticalPointSize)
		binary.PutU16(os2.usUpperOpticalPointSize)
	}

	return data
}
