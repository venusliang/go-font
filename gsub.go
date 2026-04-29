package gofont

import "errors"

// GSUB represents the Glyph Substitution table.
type GSUB struct {
	MajorVersion uint16
	MinorVersion uint16
	ScriptList   ScriptList
	FeatureList  FeatureList
	LookupList   OTLookupList
}

func parseGsub(data []byte) (*GSUB, error) {
	if len(data) < 10 {
		return nil, errors.New("GSUB table too small")
	}

	binary := BinaryFrom(data, false)
	gsub := &GSUB{
		MajorVersion: binary.U16(),
		MinorVersion: binary.U16(),
	}
	scriptListOffset := binary.U16()
	featureListOffset := binary.U16()
	lookupListOffset := binary.U16()

	gsub.ScriptList = parseScriptList(data, int(scriptListOffset))
	gsub.FeatureList = parseFeatureList(data, int(featureListOffset))
	gsub.LookupList = parseLookupList(data, int(lookupListOffset))

	return gsub, nil
}

func writeGsub(gsub *GSUB) []byte {
	// Header: majorVersion(2) + minorVersion(2) + 3 offsets(6) = 10 bytes
	headerSize := 10

	slData := writeScriptList(gsub.ScriptList)
	flData := writeFeatureList(gsub.FeatureList)
	llData := writeLookupList(gsub.LookupList)

	// Calculate offsets (all relative to table start)
	slOffset := headerSize
	flOffset := slOffset + len(slData)
	llOffset := flOffset + len(flData)

	totalSize := llOffset + len(llData)
	data := make([]byte, totalSize)
	binary := BinaryFrom(data, false)

	binary.PutU16(gsub.MajorVersion)
	binary.PutU16(gsub.MinorVersion)
	binary.PutU16(uint16(slOffset))
	binary.PutU16(uint16(flOffset))
	binary.PutU16(uint16(llOffset))

	binary.Append(slData)
	binary.Append(flData)
	binary.Append(llData)

	return data
}
