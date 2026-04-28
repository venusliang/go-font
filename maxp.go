package gofonts

import "errors"

type Maxp struct {
	version               uint32
	numGlyphs             uint16
	maxPoints             uint16
	maxContours           uint16
	maxCompositePoints    uint16
	maxCompositeContours  uint16
	maxZones              uint16
	maxTwilightPoints     uint16
	maxStorage            uint16
	maxFunctionDefs       uint16
	maxInstructionDefs    uint16
	maxStackElements      uint16
	maxSizeOfInstructions uint16
	maxComponentElements  uint16
	maxComponentDepth     uint16
}

func parseMaxp(data []byte) (*Maxp, error) {
	if len(data) < 32 {
		return nil, errors.New("maxp table too short")
	}

	binary := BinaryFrom(data, false)

	maxp := &Maxp{
		version:               binary.U32(),
		numGlyphs:             binary.U16(),
		maxPoints:             binary.U16(),
		maxContours:           binary.U16(),
		maxCompositePoints:    binary.U16(),
		maxCompositeContours:  binary.U16(),
		maxZones:              binary.U16(),
		maxTwilightPoints:     binary.U16(),
		maxStorage:            binary.U16(),
		maxFunctionDefs:       binary.U16(),
		maxInstructionDefs:    binary.U16(),
		maxStackElements:      binary.U16(),
		maxSizeOfInstructions: binary.U16(),
		maxComponentElements:  binary.U16(),
		maxComponentDepth:     binary.U16(),
	}

	return maxp, nil
}

func writeMaxp(maxp *Maxp) []byte {
	data := make([]byte, 32)
	binary := BinaryFrom(data, false)

	binary.PutU32(maxp.version)
	binary.PutU16(maxp.numGlyphs)
	binary.PutU16(maxp.maxPoints)
	binary.PutU16(maxp.maxContours)
	binary.PutU16(maxp.maxCompositePoints)
	binary.PutU16(maxp.maxCompositeContours)
	binary.PutU16(maxp.maxZones)
	binary.PutU16(maxp.maxTwilightPoints)
	binary.PutU16(maxp.maxStorage)
	binary.PutU16(maxp.maxFunctionDefs)
	binary.PutU16(maxp.maxInstructionDefs)
	binary.PutU16(maxp.maxStackElements)
	binary.PutU16(maxp.maxSizeOfInstructions)
	binary.PutU16(maxp.maxComponentElements)
	binary.PutU16(maxp.maxComponentDepth)

	return data
}
