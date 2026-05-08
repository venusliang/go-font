package gofont

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
	if len(data) < 6 {
		return nil, errors.New("maxp table too short")
	}

	binary := BinaryFrom(data, false)

	maxp := &Maxp{
		version:   binary.U32(),
		numGlyphs: binary.U16(),
	}

	// CFF fonts (version 0x00005000) only have version + numGlyphs (6 bytes).
	// TrueType fonts (version 0x00010000) have the full 32-byte structure.
	if maxp.version == 0x00010000 && len(data) >= 32 {
		maxp.maxPoints = binary.U16()
		maxp.maxContours = binary.U16()
		maxp.maxCompositePoints = binary.U16()
		maxp.maxCompositeContours = binary.U16()
		maxp.maxZones = binary.U16()
		maxp.maxTwilightPoints = binary.U16()
		maxp.maxStorage = binary.U16()
		maxp.maxFunctionDefs = binary.U16()
		maxp.maxInstructionDefs = binary.U16()
		maxp.maxStackElements = binary.U16()
		maxp.maxSizeOfInstructions = binary.U16()
		maxp.maxComponentElements = binary.U16()
		maxp.maxComponentDepth = binary.U16()
	}

	return maxp, nil
}

func writeMaxp(maxp *Maxp) []byte {
	// CFF fonts use 6-byte maxp (version 0x00005000)
	if maxp.version != 0x00010000 {
		data := make([]byte, 6)
		binary := BinaryFrom(data, false)
		binary.PutU32(maxp.version)
		binary.PutU16(maxp.numGlyphs)
		return data
	}

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
