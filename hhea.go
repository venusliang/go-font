package gofont

import "errors"

type Hhea struct {
	version             uint32
	ascent              int16
	descent             int16
	lineGap             int16
	advanceWidthMax     uint16
	minLeftSideBearing  int16
	minRightSideBearing int16
	xMaxExtent          int16
	caretSlopeRise      int16
	caretSlopeRun       int16
	caretOffset         int16
	reserved1           int16
	reserved2           int16
	reserved3           int16
	reserved4           int16
	metricDataFormat    int16
	numberOfHMetrics    uint16
}

func parseHhea(data []byte) (*Hhea, error) {
	if len(data) < 36 {
		return nil, errors.New("hhea table too small")
	}

	binary := BinaryFrom(data, false)

	hhea := &Hhea{
		version:             binary.U32(),
		ascent:              binary.I16(),
		descent:             binary.I16(),
		lineGap:             binary.I16(),
		advanceWidthMax:     binary.U16(),
		minLeftSideBearing:  binary.I16(),
		minRightSideBearing: binary.I16(),
		xMaxExtent:          binary.I16(),
		caretSlopeRise:      binary.I16(),
		caretSlopeRun:       binary.I16(),
		caretOffset:         binary.I16(),
		reserved1:           binary.I16(),
		reserved2:           binary.I16(),
		reserved3:           binary.I16(),
		reserved4:           binary.I16(),
		metricDataFormat:    binary.I16(),
		numberOfHMetrics:    binary.U16(),
	}

	return hhea, nil
}

func writeHhea(hhea *Hhea) []byte {
	data := make([]byte, 36)
	binary := BinaryFrom(data, false)

	binary.PutU32(hhea.version)
	binary.PutU16(uint16(hhea.ascent))
	binary.PutU16(uint16(hhea.descent))
	binary.PutU16(uint16(hhea.lineGap))
	binary.PutU16(hhea.advanceWidthMax)
	binary.PutU16(uint16(hhea.minLeftSideBearing))
	binary.PutU16(uint16(hhea.minRightSideBearing))
	binary.PutU16(uint16(hhea.xMaxExtent))
	binary.PutU16(uint16(hhea.caretSlopeRise))
	binary.PutU16(uint16(hhea.caretSlopeRun))
	binary.PutU16(uint16(hhea.caretOffset))
	binary.PutU16(uint16(hhea.reserved1))
	binary.PutU16(uint16(hhea.reserved2))
	binary.PutU16(uint16(hhea.reserved3))
	binary.PutU16(uint16(hhea.reserved4))
	binary.PutU16(uint16(hhea.metricDataFormat))
	binary.PutU16(hhea.numberOfHMetrics)

	return data
}
