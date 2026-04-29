package gofont

// --- Coverage table ---

// Coverage represents a glyph coverage table (format 1 or 2).
type Coverage struct {
	Format uint16           // 1 = glyph list, 2 = ranges
	Glyphs []uint16         // Format 1
	Ranges []CoverageRange  // Format 2
}

// CoverageRange represents a range record in Coverage format 2.
type CoverageRange struct {
	Start uint16 // startGlyphID
	End   uint16 // endGlyphID
	Index uint16 // startCoverageIndex
}

func parseCoverage(data []byte, offset int) Coverage {
	if offset >= len(data) {
		return Coverage{}
	}
	binary := BinaryFrom(data[offset:], false)
	cov := Coverage{Format: binary.U16()}

	switch cov.Format {
	case 1:
		count := binary.U16()
		cov.Glyphs = make([]uint16, count)
		for i := range cov.Glyphs {
			cov.Glyphs[i] = binary.U16()
		}
	case 2:
		count := binary.U16()
		cov.Ranges = make([]CoverageRange, count)
		for i := range cov.Ranges {
			cov.Ranges[i] = CoverageRange{
				Start: binary.U16(),
				End:   binary.U16(),
				Index: binary.U16(),
			}
		}
	}
	return cov
}

func writeCoverage(cov Coverage) []byte {
	switch cov.Format {
	case 1:
		data := make([]byte, 4+len(cov.Glyphs)*2)
		binary := BinaryFrom(data, false)
		binary.PutU16(1)
		binary.PutU16(uint16(len(cov.Glyphs)))
		for _, g := range cov.Glyphs {
			binary.PutU16(g)
		}
		return data
	case 2:
		data := make([]byte, 4+len(cov.Ranges)*6)
		binary := BinaryFrom(data, false)
		binary.PutU16(2)
		binary.PutU16(uint16(len(cov.Ranges)))
		for _, r := range cov.Ranges {
			binary.PutU16(r.Start)
			binary.PutU16(r.End)
			binary.PutU16(r.Index)
		}
		return data
	default:
		return nil
	}
}

// --- ClassDef table ---

// ClassDef represents a class definition table (format 1 or 2).
type ClassDef struct {
	Format      uint16             // 1 = sequential, 2 = ranges
	StartGlyph  uint16             // Format 1
	ClassValues []uint16           // Format 1
	Ranges      []ClassRangeRecord // Format 2
}

// ClassRangeRecord represents a range record in ClassDef format 2.
type ClassRangeRecord struct {
	Start uint16 // startGlyphID
	End   uint16 // endGlyphID
	Class uint16 // classValue
}

func parseClassDef(data []byte, offset int) ClassDef {
	if offset >= len(data) {
		return ClassDef{}
	}
	binary := BinaryFrom(data[offset:], false)
	cd := ClassDef{Format: binary.U16()}

	switch cd.Format {
	case 1:
		cd.StartGlyph = binary.U16()
		count := binary.U16()
		cd.ClassValues = make([]uint16, count)
		for i := range cd.ClassValues {
			cd.ClassValues[i] = binary.U16()
		}
	case 2:
		count := binary.U16()
		cd.Ranges = make([]ClassRangeRecord, count)
		for i := range cd.Ranges {
			cd.Ranges[i] = ClassRangeRecord{
				Start: binary.U16(),
				End:   binary.U16(),
				Class: binary.U16(),
			}
		}
	}
	return cd
}

func writeClassDef(cd ClassDef) []byte {
	switch cd.Format {
	case 1:
		data := make([]byte, 6+len(cd.ClassValues)*2)
		binary := BinaryFrom(data, false)
		binary.PutU16(1)
		binary.PutU16(cd.StartGlyph)
		binary.PutU16(uint16(len(cd.ClassValues)))
		for _, v := range cd.ClassValues {
			binary.PutU16(v)
		}
		return data
	case 2:
		data := make([]byte, 4+len(cd.Ranges)*6)
		binary := BinaryFrom(data, false)
		binary.PutU16(2)
		binary.PutU16(uint16(len(cd.Ranges)))
		for _, r := range cd.Ranges {
			binary.PutU16(r.Start)
			binary.PutU16(r.End)
			binary.PutU16(r.Class)
		}
		return data
	default:
		return nil
	}
}

// --- Script List ---

// ScriptList represents the list of scripts in an OpenType Layout table.
type ScriptList struct {
	Records []ScriptRecord
}

// ScriptRecord maps a script tag to its script table.
type ScriptRecord struct {
	Tag    [4]byte
	Script ScriptTable
}

// ScriptTable represents a script table with optional default LangSys and language-specific records.
type ScriptTable struct {
	DefaultLangSys *LangSys
	LangSysRecords []LangSysRecord
}

// LangSysRecord maps a language tag to its LangSys table.
type LangSysRecord struct {
	Tag     [4]byte
	LangSys LangSys
}

// LangSys represents a language system table.
type LangSys struct {
	ReqFeatureIndex uint16
	FeatureIndices  []uint16
}

func parseScriptList(data []byte, offset int) ScriptList {
	if offset >= len(data) {
		return ScriptList{}
	}
	binary := BinaryFrom(data[offset:], false)
	count := binary.U16()
	sl := ScriptList{Records: make([]ScriptRecord, count)}

	for i := range sl.Records {
		copy(sl.Records[i].Tag[:], binary.Read(4))
		scriptOffset := binary.U16()
		sl.Records[i].Script = parseScript(data, offset+int(scriptOffset))
	}
	return sl
}

func parseScript(data []byte, offset int) ScriptTable {
	if offset >= len(data) {
		return ScriptTable{}
	}
	binary := BinaryFrom(data[offset:], false)
	defaultLangSysOffset := binary.U16()
	langSysCount := binary.U16()

	st := ScriptTable{}
	if defaultLangSysOffset != 0 {
		defLS := parseLangSys(data, offset+int(defaultLangSysOffset))
		st.DefaultLangSys = &defLS
	}

	st.LangSysRecords = make([]LangSysRecord, langSysCount)
	for i := range st.LangSysRecords {
		copy(st.LangSysRecords[i].Tag[:], binary.Read(4))
		lsOffset := binary.U16()
		st.LangSysRecords[i].LangSys = parseLangSys(data, offset+int(lsOffset))
	}
	return st
}

func parseLangSys(data []byte, offset int) LangSys {
	if offset >= len(data) {
		return LangSys{}
	}
	binary := BinaryFrom(data[offset:], false)
	ls := LangSys{}
	_ = binary.U16() // lookupOrderOffset (reserved)
	ls.ReqFeatureIndex = binary.U16()
	featureCount := binary.U16()
	ls.FeatureIndices = make([]uint16, featureCount)
	for i := range ls.FeatureIndices {
		ls.FeatureIndices[i] = binary.U16()
	}
	return ls
}

func writeScriptList(sl ScriptList) []byte {
	// Strategy: serialize each ScriptTable, then assemble with offsets.
	// ScriptList header: uint16 count + count * {4-byte tag + uint16 offset}
	headerSize := 2 + len(sl.Records)*6

	// Serialize each script table
	scriptDatas := make([][]byte, len(sl.Records))
	for i, rec := range sl.Records {
		scriptDatas[i] = writeScriptTable(rec.Script)
	}

	// Calculate offsets
	offsets := make([]int, len(sl.Records))
	off := headerSize
	for i, sd := range scriptDatas {
		offsets[i] = off
		off += len(sd)
	}

	// Build result
	totalSize := off
	data := make([]byte, totalSize)
	binary := BinaryFrom(data, false)
	binary.PutU16(uint16(len(sl.Records)))
	for i, rec := range sl.Records {
		binary.Append(rec.Tag[:])
		binary.PutU16(uint16(offsets[i]))
	}
	for _, sd := range scriptDatas {
		binary.Append(sd)
	}
	return data
}

func writeScriptTable(st ScriptTable) []byte {
	// Header: uint16 defaultLangSysOffset + uint16 langSysCount + count * {4-byte tag + uint16 offset}
	langCount := len(st.LangSysRecords)
	headerSize := 4 + langCount*6

	// Serialize default LangSys
	var defaultLSData []byte
	if st.DefaultLangSys != nil {
		defaultLSData = writeLangSys(*st.DefaultLangSys)
	}

	// Serialize each LangSys
	lsDatas := make([][]byte, langCount)
	for i, rec := range st.LangSysRecords {
		lsDatas[i] = writeLangSys(rec.LangSys)
	}

	// Calculate offsets
	defaultLSOffset := uint16(0)
	off := headerSize
	if defaultLSData != nil {
		defaultLSOffset = uint16(off)
		off += len(defaultLSData)
	}

	lsOffsets := make([]int, langCount)
	for i, lsd := range lsDatas {
		lsOffsets[i] = off
		off += len(lsd)
	}

	// Build result
	data := make([]byte, off)
	binary := BinaryFrom(data, false)
	binary.PutU16(defaultLSOffset)
	binary.PutU16(uint16(langCount))
	for i, rec := range st.LangSysRecords {
		binary.Append(rec.Tag[:])
		binary.PutU16(uint16(lsOffsets[i]))
	}
	if defaultLSData != nil {
		binary.Append(defaultLSData)
	}
	for _, lsd := range lsDatas {
		binary.Append(lsd)
	}
	return data
}

func writeLangSys(ls LangSys) []byte {
	data := make([]byte, 6+len(ls.FeatureIndices)*2)
	binary := BinaryFrom(data, false)
	binary.PutU16(0) // lookupOrderOffset (reserved)
	binary.PutU16(ls.ReqFeatureIndex)
	binary.PutU16(uint16(len(ls.FeatureIndices)))
	for _, fi := range ls.FeatureIndices {
		binary.PutU16(fi)
	}
	return data
}

// --- Feature List ---

// FeatureList represents the list of features in an OpenType Layout table.
type FeatureList struct {
	Records []FeatureRecord
}

// FeatureRecord maps a feature tag to its feature table.
type FeatureRecord struct {
	Tag     [4]byte
	Feature FeatureTable
}

// FeatureTable represents a feature table.
type FeatureTable struct {
	FeatureParams uint16
	LookupIndices []uint16
}

func parseFeatureList(data []byte, offset int) FeatureList {
	if offset >= len(data) {
		return FeatureList{}
	}
	binary := BinaryFrom(data[offset:], false)
	count := binary.U16()
	fl := FeatureList{Records: make([]FeatureRecord, count)}

	for i := range fl.Records {
		copy(fl.Records[i].Tag[:], binary.Read(4))
		featureOffset := binary.U16()
		fl.Records[i].Feature = parseFeature(data, offset+int(featureOffset))
	}
	return fl
}

func parseFeature(data []byte, offset int) FeatureTable {
	if offset >= len(data) {
		return FeatureTable{}
	}
	binary := BinaryFrom(data[offset:], false)
	ft := FeatureTable{
		FeatureParams: binary.U16(),
	}
	count := binary.U16()
	ft.LookupIndices = make([]uint16, count)
	for i := range ft.LookupIndices {
		ft.LookupIndices[i] = binary.U16()
	}
	return ft
}

func writeFeatureList(fl FeatureList) []byte {
	// Header: uint16 count + count * {4-byte tag + uint16 offset}
	headerSize := 2 + len(fl.Records)*6

	featureDatas := make([][]byte, len(fl.Records))
	for i, rec := range fl.Records {
		featureDatas[i] = writeFeatureTable(rec.Feature)
	}

	offsets := make([]int, len(fl.Records))
	off := headerSize
	for i, fd := range featureDatas {
		offsets[i] = off
		off += len(fd)
	}

	data := make([]byte, off)
	binary := BinaryFrom(data, false)
	binary.PutU16(uint16(len(fl.Records)))
	for i, rec := range fl.Records {
		binary.Append(rec.Tag[:])
		binary.PutU16(uint16(offsets[i]))
	}
	for _, fd := range featureDatas {
		binary.Append(fd)
	}
	return data
}

func writeFeatureTable(ft FeatureTable) []byte {
	data := make([]byte, 4+len(ft.LookupIndices)*2)
	binary := BinaryFrom(data, false)
	binary.PutU16(ft.FeatureParams)
	binary.PutU16(uint16(len(ft.LookupIndices)))
	for _, li := range ft.LookupIndices {
		binary.PutU16(li)
	}
	return data
}

// --- Lookup List ---

// OTLookupList represents the list of lookups in an OpenType Layout table.
type OTLookupList struct {
	Lookups []OTLookup
}

// OTLookup represents a single lookup table.
type OTLookup struct {
	LookupType      uint16
	LookupFlag      uint16
	MarkFilteringSet uint16
	SubTables       [][]byte // raw subtable data
}

func parseLookupList(data []byte, offset int) OTLookupList {
	if offset >= len(data) {
		return OTLookupList{}
	}
	binary := BinaryFrom(data[offset:], false)
	count := binary.U16()
	ll := OTLookupList{Lookups: make([]OTLookup, count)}

	// Read all lookup offsets first
	lookupOffsets := make([]uint16, count)
	for i := range lookupOffsets {
		lookupOffsets[i] = binary.U16()
	}

	// Parse each lookup, passing the end boundary for each
	for i := range ll.Lookups {
		absOffset := offset + int(lookupOffsets[i])
		var lookupEnd int
		if i+1 < int(count) {
			lookupEnd = offset + int(lookupOffsets[i+1])
		} else {
			lookupEnd = len(data)
		}
		ll.Lookups[i] = parseLookup(data, absOffset, lookupEnd)
	}

	return ll
}

func parseLookup(data []byte, offset int, lookupEnd int) OTLookup {
	if offset+6 > len(data) {
		return OTLookup{}
	}
	binary := BinaryFrom(data[offset:], false)
	lk := OTLookup{
		LookupType: binary.U16(),
		LookupFlag: binary.U16(),
	}
	subTableCount := binary.U16()

	// Read subtable offsets (relative to lookup start)
	subOffsets := make([]uint16, subTableCount)
	for i := range subOffsets {
		subOffsets[i] = binary.U16()
	}

	// Read markFilteringSet if flag bit 4 is set
	if lk.LookupFlag&0x0010 != 0 {
		lk.MarkFilteringSet = binary.U16()
	}

	// Calculate lookup header size for offset calculation
	headerSize := 6 + int(subTableCount)*2
	if lk.LookupFlag&0x0010 != 0 {
		headerSize += 2
	}

	// Extract each subtable's raw data
	// Sort offsets to calculate sizes
	sortedOffsets := make([]int, subTableCount)
	for i, o := range subOffsets {
		sortedOffsets[i] = int(o)
	}
	// Simple approach: extract bytes between consecutive subtable offsets
	// First, determine lookup end by looking at all subtable starts
	// The lookup data extends past the last subtable
	// For each subtable, we read from its offset to the next subtable's offset
	// For the last subtable, we need to determine its size from its content

	lk.SubTables = make([][]byte, subTableCount)
	for i := range lk.SubTables {
		start := int(subOffsets[i])

		var end int
		if i+1 < int(subTableCount) {
			end = int(subOffsets[i+1])
		} else {
			// Last subtable: extend to the end of this lookup's data
			end = lookupEnd - offset
		}

		if end > len(data)-offset {
			end = len(data) - offset
		}

		subLen := end - start
		if subLen < 0 {
			subLen = 0
		}
		lk.SubTables[i] = make([]byte, subLen)
		copy(lk.SubTables[i], data[offset+start:offset+start+subLen])
	}

	return lk
}

func writeLookupList(ll OTLookupList) []byte {
	// Header: uint16 count + count * uint16 offset
	headerSize := 2 + len(ll.Lookups)*2

	// Serialize each lookup
	lookupDatas := make([][]byte, len(ll.Lookups))
	for i, lk := range ll.Lookups {
		lookupDatas[i] = writeLookup(lk)
	}

	// Calculate offsets
	offsets := make([]int, len(ll.Lookups))
	off := headerSize
	for i, ld := range lookupDatas {
		offsets[i] = off
		off += len(ld)
	}

	// Build result
	data := make([]byte, off)
	binary := BinaryFrom(data, false)
	binary.PutU16(uint16(len(ll.Lookups)))
	for _, o := range offsets {
		binary.PutU16(uint16(o))
	}
	for _, ld := range lookupDatas {
		binary.Append(ld)
	}
	return data
}

func writeLookup(lk OTLookup) []byte {
	// Header: lookupType(2) + lookupFlag(2) + subTableCount(2) + subtableOffsets(n*2) + [markFilteringSet(2)]
	hasMarkFilter := lk.LookupFlag&0x0010 != 0
	headerSize := 6 + len(lk.SubTables)*2
	if hasMarkFilter {
		headerSize += 2
	}

	// Subtable data starts after header
	// Each subtable offset is relative to the start of this lookup
	subOffsets := make([]int, len(lk.SubTables))
	off := headerSize
	for i, st := range lk.SubTables {
		subOffsets[i] = off
		off += len(st)
	}

	data := make([]byte, off)
	binary := BinaryFrom(data, false)
	binary.PutU16(lk.LookupType)
	binary.PutU16(lk.LookupFlag)
	binary.PutU16(uint16(len(lk.SubTables)))
	for _, o := range subOffsets {
		binary.PutU16(uint16(o))
	}
	if hasMarkFilter {
		binary.PutU16(lk.MarkFilteringSet)
	}
	for _, st := range lk.SubTables {
		binary.Append(st)
	}
	return data
}
