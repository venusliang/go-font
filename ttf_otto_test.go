package gofont

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	ottoFontData     []byte
	ottoFontDataOnce sync.Once
)

// loadOTTOFont reads a minimal OTTO (OpenType/CFF) test font.
func loadOTTOFont(t *testing.T) []byte {
	t.Helper()
	ottoFontDataOnce.Do(func() {
		paths := []string{
			"fonts/otto-test.otf",
			filepath.Join("..", "fonts", "otto-test.otf"),
		}
		var err error
		for _, p := range paths {
			ottoFontData, err = os.ReadFile(p)
			if err == nil {
				return
			}
		}
		// If no OTF file exists, generate a minimal one programmatically
		ottoFontData = generateMinimalOTTO(t)
	})
	return ottoFontData
}

func TestParseOTTO(t *testing.T) {
	data := loadOTTOFont(t)
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse OTTO font failed: %v", err)
	}

	if !font.IsCFF() {
		t.Error("IsCFF() should return true for OTTO font")
	}

	// Verify common tables are parsed
	if font.head == nil {
		t.Error("head table should be parsed")
	}
	if font.maxp == nil {
		t.Error("maxp table should be parsed")
	}
	if font.hhea == nil {
		t.Error("hhea table should be parsed")
	}
	if font.name == nil {
		t.Error("name table should be parsed")
	}
	if font.cmap == nil {
		t.Error("cmap table should be parsed")
	}
	if font.hmtx == nil {
		t.Error("hmtx table should be parsed")
	}

	// CFF font should NOT have glyf/loca
	if font.glyf != nil {
		t.Error("CFF font should not have glyf table parsed")
	}
	if font.loca != nil {
		t.Error("CFF font should not have loca table parsed")
	}

	// maxp version should be 0x00005000 for CFF
	if font.maxp.version != 0x00005000 {
		t.Errorf("maxp version = 0x%08X, want 0x00005000", font.maxp.version)
	}
	if font.maxp.numGlyphs == 0 {
		t.Error("maxp.numGlyphs should be > 0")
	}

	// CFF raw table data should be preserved
	if len(font.rawTables) == 0 {
		t.Error("rawTables should contain CFF table data")
	}
	hasCFF := false
	for tag := range font.rawTables {
		if tag == "CFF " {
			hasCFF = true
			break
		}
	}
	if !hasCFF {
		t.Error("rawTables should contain 'CFF ' table")
	}
}

func TestRoundTripOTTO(t *testing.T) {
	data := loadOTTOFont(t)
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse OTTO font failed: %v", err)
	}

	serialized, err := font.Serialize()
	if err != nil {
		t.Fatalf("Serialize OTTO font failed: %v", err)
	}

	font2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("Re-parse serialized OTTO font failed: %v", err)
	}

	// Verify version preserved
	if font2.version != font.version {
		t.Errorf("version mismatch: got 0x%08X, want 0x%08X", font2.version, font.version)
	}

	// Verify key fields match
	if font2.head.unitsPerEm != font.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: got %d, want %d", font2.head.unitsPerEm, font.head.unitsPerEm)
	}
	if font2.maxp.numGlyphs != font.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: got %d, want %d", font2.maxp.numGlyphs, font.maxp.numGlyphs)
	}
	if font2.hhea.ascent != font.hhea.ascent {
		t.Errorf("ascent mismatch: got %d, want %d", font2.hhea.ascent, font.hhea.ascent)
	}

	// CFF raw table should survive round-trip
	if _, ok := font2.rawTables["CFF "]; !ok {
		t.Error("CFF table should survive round-trip")
	}
}

func TestOTTOFontMetrics(t *testing.T) {
	data := loadOTTOFont(t)
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse OTTO font failed: %v", err)
	}

	// UnitsPerEm should work
	upm := font.UnitsPerEm()
	if upm == 0 {
		t.Error("UnitsPerEm should be > 0")
	}

	// FontBBox should work
	xMin, yMin, xMax, yMax := font.FontBBox()
	_ = xMin
	_ = yMin
	_ = xMax
	_ = yMax

	// Ascent/Descent should work
	ascent := font.Ascent()
	if ascent == 0 {
		t.Error("Ascent should be non-zero")
	}

	// NumGlyphs returns len(glyf) which is 0 for CFF fonts
	// This is expected behavior for Phase 1
	numGlyphs := font.NumGlyphs()
	if numGlyphs != 0 {
		t.Errorf("NumGlyphs for CFF font (Phase 1) should be 0, got %d", numGlyphs)
	}
}

// generateMinimalOTTO creates a minimal valid OTTO font file for testing.
func generateMinimalOTTO(t *testing.T) []byte {
	t.Helper()

	// head table (54 bytes)
	headData := make([]byte, 54)
	b := BinaryFrom(headData, false)
	b.PutU16(1)            // majorVersion
	b.PutU16(0)            // minorVersion
	b.PutU16(1)            // fontRevision major
	b.PutU16(0)            // fontRevision minor
	b.PutU32(0)            // checksumAdjustment (patched later)
	b.PutU32(0x5F0F3CF5)   // magicNumber
	b.PutU16(0x000B)       // flags
	b.PutU16(1000)         // unitsPerEm
	b.PutU64(0)            // created
	b.PutU64(0)            // modified
	var neg200 int16 = -200
	b.PutU16(0)                // xMin
	b.PutU16(uint16(neg200))   // yMin
	b.PutU16(uint16(600))      // xMax
	b.PutU16(uint16(800))      // yMax
	b.PutU16(0)            // macStyle
	b.PutU16(8)            // lowestRecPPEM
	b.PutU16(uint16(2))    // fontDirectionHint
	b.PutU16(0)            // indexToLocFormat
	b.PutU16(0)            // glyphDataFormat

	// hhea table (36 bytes)
	hheaData := make([]byte, 36)
	b = BinaryFrom(hheaData, false)
	b.PutU32(0x00010000)         // version
	b.PutU16(uint16(int16(800))) // ascent
	b.PutU16(uint16(neg200))    // descent
	b.PutU16(0)              // lineGap
	b.PutU16(600)            // advanceWidthMax
	b.PutU16(0)              // minLeftSideBearing
	b.PutU16(0)              // minRightSideBearing
	b.PutU16(uint16(800))    // xMaxExtent
	b.PutU16(uint16(1))      // caretSlopeRise
	b.PutU16(0)              // caretSlopeRun
	b.PutU16(0)              // caretOffset
	b.PutU16(0)              // reserved1
	b.PutU16(0)              // reserved2
	b.PutU16(0)              // reserved3
	b.PutU16(0)              // reserved4
	b.PutU16(0)              // metricDataFormat
	b.PutU16(2)              // numberOfHMetrics

	// maxp table (6 bytes for CFF)
	maxpData := make([]byte, 6)
	b = BinaryFrom(maxpData, false)
	b.PutU32(0x00005000) // CFF version
	b.PutU16(2)          // numGlyphs

	// hmtx table (2 glyphs × 4 bytes = 8 bytes)
	hmtxData := make([]byte, 8)
	b = BinaryFrom(hmtxData, false)
	b.PutU16(500) // advanceWidth glyph 0
	b.PutU16(0)   // lsb glyph 0
	b.PutU16(500) // advanceWidth glyph 1
	b.PutU16(0)   // lsb glyph 1

	// name table
	familyName := encodeUTF16BE("TestOTTO")
	fullName := encodeUTF16BE("Test OTTO Font")
	psName := encodeUTF16BE("TestOTTO")
	type nameRecData struct {
		platformID, encodingID, languageID, nameID uint16
		data                                       []byte
	}
	nrData := []nameRecData{
		{3, 1, 0x0409, 1, familyName},
		{3, 1, 0x0409, 4, fullName},
		{3, 1, 0x0409, 6, psName},
	}
	stringStorage := make([]byte, 0)
	off := uint16(0)
	type nameRec struct {
		platformID, encodingID, languageID, nameID, length, offset uint16
	}
	records := make([]nameRec, len(nrData))
	for i, nr := range nrData {
		records[i] = nameRec{nr.platformID, nr.encodingID, nr.languageID, nr.nameID, uint16(len(nr.data)), off}
		stringStorage = append(stringStorage, nr.data...)
		off += uint16(len(nr.data))
	}
	nameData := make([]byte, 6+12*len(records)+len(stringStorage))
	b = BinaryFrom(nameData, false)
	b.PutU16(0)                              // format
	b.PutU16(uint16(len(records)))           // count
	b.PutU16(6 + uint16(12*len(records)))    // stringOffset
	for _, r := range records {
		b.PutU16(r.platformID)
		b.PutU16(r.encodingID)
		b.PutU16(r.languageID)
		b.PutU16(r.nameID)
		b.PutU16(r.length)
		b.PutU16(r.offset)
	}
	b.Append(stringStorage)

	// cmap table - format 4 mapping 'A' (U+0041) -> glyph 1
	segCount := uint16(2)
	cmapSubtable := make([]byte, 14+int(segCount)*8+2) // +2 for reservedPad
	b = BinaryFrom(cmapSubtable, false)
	b.PutU16(4)                          // format
	b.PutU16(0)                          // length (patched below)
	b.PutU16(0)                          // language
	b.PutU16(segCount * 2)               // segCountX2
	sr, es, rs := calcSearchParams(int(segCount))
	b.PutU16(sr)
	b.PutU16(es)
	b.PutU16(rs)
	// endCode
	b.PutU16(0x0041)
	b.PutU16(0xFFFF)
	// reservedPad
	b.PutU16(0)
	// startCode
	b.PutU16(0x0041)
	b.PutU16(0xFFFF)
	// idDelta
	var delta int16 = 1 - 0x0041
	b.PutU16(uint16(delta))
	b.PutU16(uint16(int16(1)))
	// idRangeOffset
	b.PutU16(0)
	b.PutU16(0)
	// Patch length at offset 2
	b2 := BinaryFrom(cmapSubtable[2:4], false)
	b2.PutU16(uint16(len(cmapSubtable)))

	cmapData := make([]byte, 4+8+len(cmapSubtable))
	b = BinaryFrom(cmapData, false)
	b.PutU16(0)  // version
	b.PutU16(1)  // numTables
	b.PutU16(3)  // platformID (Windows)
	b.PutU16(1)  // encodingID (Unicode BMP)
	b.PutU32(12) // offset to subtable
	b.Append(cmapSubtable)

	// OS/2 table (86 bytes for version 1: 78 base + 4 ulCodePageRange1 + 4 ulCodePageRange2)
	os2Data := make([]byte, 86)
	b = BinaryFrom(os2Data, false)
	b.PutU16(1)          // version
	b.PutU16(uint16(500)) // xAvgCharWidth
	b.PutU16(400)        // usWeightClass
	b.PutU16(5)          // usWidthClass
	b.PutU16(0)          // fsType
	for i := 0; i < 10; i++ { // ySubscript/YSuperscript fields (10 × int16)
		b.PutU16(0)
	}
	b.Append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) // panose
	b.Append([]byte{0, 0, 0, 0}) // ulUnicodeRange1
	b.Append([]byte{0, 0, 0, 0}) // ulUnicodeRange2
	b.Append([]byte{0, 0, 0, 0}) // ulUnicodeRange3
	b.Append([]byte{0, 0, 0, 4}) // ulUnicodeRange4
	b.Append([]byte("    "))     // achVendID
	b.PutU16(0x0040)             // fsSelection
	b.PutU16(0x0041)             // usFirstCharIndex
	b.PutU16(0x0041)             // usLastCharIndex
	b.PutU16(uint16(800))        // sTypoAscender
	b.PutU16(uint16(neg200))          // sTypoDescender
	b.PutU16(0)                  // sTypoLineGap
	b.PutU16(800)                // usWinAscent
	b.PutU16(200)                // usWinDescent

	// CFF table - minimal valid CFF data
	cffData := []byte{
		1, 0, 4, 1,                       // header: major=1, minor=0, hdrSize=4, offSize=1
		0, 1, 1, 1, 9,                    // Name INDEX: count=1, offSize=1, offsets=[1,9]
		'T', 'e', 's', 't', 'O', 'T', 'T', 'O',
		0, 1, 1, 1, 2,                    // Top DICT INDEX: count=1, offSize=1, offsets=[1,2]
		0x8B,                             // Top DICT: charset=0 (ISOAdobe)
		0, 0,                             // String INDEX: empty
		0, 0,                             // Global Subr INDEX: empty
	}

	// post table (format 3.0 - 32 bytes)
	postData := make([]byte, 32)
	b = BinaryFrom(postData, false)
	var neg100 int16 = -100
	b.PutU32(0x00030000)   // format 3.0
	b.PutU16(0)            // italicAngle (fixed)
	b.PutU16(0)            // italicAngle frac
	b.PutU16(uint16(neg100)) // underlinePosition
	b.PutU16(uint16(50))     // underlineThickness
	b.PutU32(0)            // isFixedPitch
	b.PutU32(0)            // minMemType42
	b.PutU32(0)            // maxMemType42
	b.PutU32(0)            // minMemType1
	b.PutU32(0)            // maxMemType1

	// Assemble the font file
	type tableEntry struct {
		tag  string
		data []byte
	}
	tables := []tableEntry{
		{"CFF ", cffData},
		{"OS/2", os2Data},
		{"cmap", cmapData},
		{"head", headData},
		{"hhea", hheaData},
		{"hmtx", hmtxData},
		{"maxp", maxpData},
		{"name", nameData},
		{"post", postData},
	}

	numTables := len(tables)
	headerSize := 12 + 16*numTables
	offset := uint32(headerSize)

	type tableOffset struct {
		tag      string
		data     []byte
		offset   uint32
		length   uint32
		checksum uint32
	}
	tableOffsets := make([]tableOffset, numTables)

	for i, tbl := range tables {
		if offset%4 != 0 {
			offset += 4 - offset%4
		}
		checksumData := tbl.data
		if tbl.tag == "head" && len(tbl.data) >= 12 {
			checksumData = make([]byte, len(tbl.data))
			copy(checksumData, tbl.data)
			checksumData[8] = 0
			checksumData[9] = 0
			checksumData[10] = 0
			checksumData[11] = 0
		}
		tableOffsets[i] = tableOffset{
			tag:      tbl.tag,
			data:     tbl.data,
			offset:   offset,
			length:   uint32(len(tbl.data)),
			checksum: calcTableChecksum(checksumData),
		}
		offset += uint32(len(tbl.data))
	}

	totalSize := offset
	fileData := make([]byte, totalSize)
	b = BinaryFrom(fileData, false)

	b.PutU32(0x4F54544F) // "OTTO"
	sr2, es2, rs2 := calcSearchParams(numTables)
	b.PutU16(uint16(numTables))
	b.PutU16(sr2)
	b.PutU16(es2)
	b.PutU16(rs2)

	for _, to := range tableOffsets {
		tagBytes := []byte(to.tag)
		if len(tagBytes) < 4 {
			padded := make([]byte, 4)
			copy(padded, tagBytes)
			tagBytes = padded
		}
		b.Append(tagBytes[:4])
		b.PutU32(to.checksum)
		b.PutU32(to.offset)
		b.PutU32(to.length)
	}

	for _, to := range tableOffsets {
		copy(fileData[to.offset:], to.data)
	}

	// Patch head.checksumAdjustment
	for _, to := range tableOffsets {
		if to.tag == "head" && len(to.data) >= 12 {
			headStart := to.offset
			fileData[headStart+8] = 0
			fileData[headStart+9] = 0
			fileData[headStart+10] = 0
			fileData[headStart+11] = 0
			wholeChecksum := calcTableChecksum(fileData)
			adjustment := 0xB1B0AFBA - wholeChecksum
			bb := BinaryFrom(fileData[headStart+8:headStart+12], false)
			bb.PutU32(adjustment)
			break
		}
	}

	return fileData
}
