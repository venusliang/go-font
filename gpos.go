package gofont

import "errors"

// GPOS represents the Glyph Positioning table.
type GPOS struct {
	MajorVersion uint16
	MinorVersion uint16
	ScriptList   ScriptList
	FeatureList  FeatureList
	LookupList   OTLookupList
}

func parseGpos(data []byte) (*GPOS, error) {
	if len(data) < 10 {
		return nil, errors.New("GPOS table too small")
	}

	binary := BinaryFrom(data, false)
	gpos := &GPOS{
		MajorVersion: binary.U16(),
		MinorVersion: binary.U16(),
	}
	scriptListOffset := binary.U16()
	featureListOffset := binary.U16()
	lookupListOffset := binary.U16()

	gpos.ScriptList = parseScriptList(data, int(scriptListOffset))
	gpos.FeatureList = parseFeatureList(data, int(featureListOffset))
	gpos.LookupList = parseLookupList(data, int(lookupListOffset))

	return gpos, nil
}

func writeGpos(gpos *GPOS) []byte {
	// Header: majorVersion(2) + minorVersion(2) + 3 offsets(6) = 10 bytes
	headerSize := 10

	slData := writeScriptList(gpos.ScriptList)
	flData := writeFeatureList(gpos.FeatureList)
	llData := writeLookupList(gpos.LookupList)

	// Calculate offsets (all relative to table start)
	slOffset := headerSize
	flOffset := slOffset + len(slData)
	llOffset := flOffset + len(flData)

	totalSize := llOffset + len(llData)
	data := make([]byte, totalSize)
	binary := BinaryFrom(data, false)

	binary.PutU16(gpos.MajorVersion)
	binary.PutU16(gpos.MinorVersion)
	binary.PutU16(uint16(slOffset))
	binary.PutU16(uint16(flOffset))
	binary.PutU16(uint16(llOffset))

	binary.Append(slData)
	binary.Append(flData)
	binary.Append(llData)

	return data
}
