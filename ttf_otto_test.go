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
			"testdata/otto-test.otf",
			filepath.Join("..", "testdata", "otto-test.otf"),
		}
		var err error
		for _, p := range paths {
			ottoFontData, err = os.ReadFile(p)
			if err == nil {
				return
			}
		}
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

	if font.glyf != nil {
		t.Error("CFF font should not have glyf table parsed")
	}
	if font.loca != nil {
		t.Error("CFF font should not have loca table parsed")
	}

	if font.maxp.version != 0x00005000 {
		t.Errorf("maxp version = 0x%08X, want 0x00005000", font.maxp.version)
	}
	if font.maxp.numGlyphs == 0 {
		t.Error("maxp.numGlyphs should be > 0")
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

	if font2.version != font.version {
		t.Errorf("version mismatch: got 0x%08X, want 0x%08X", font2.version, font.version)
	}
	if font2.head.unitsPerEm != font.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: got %d, want %d", font2.head.unitsPerEm, font.head.unitsPerEm)
	}
	if font2.maxp.numGlyphs != font.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: got %d, want %d", font2.maxp.numGlyphs, font.maxp.numGlyphs)
	}
	if font2.hhea.ascent != font.hhea.ascent {
		t.Errorf("ascent mismatch: got %d, want %d", font2.hhea.ascent, font.hhea.ascent)
	}

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

	upm := font.UnitsPerEm()
	if upm == 0 {
		t.Error("UnitsPerEm should be > 0")
	}

	ascent := font.Ascent()
	if ascent == 0 {
		t.Error("Ascent should be non-zero")
	}

	// NumGlyphs should use maxp.numGlyphs for CFF fonts
	numGlyphs := font.NumGlyphs()
	if numGlyphs != 2 {
		t.Errorf("NumGlyphs for CFF font should be 2, got %d", numGlyphs)
	}

	// AdvanceWidth should work for CFF fonts
	aw := font.AdvanceWidth(0)
	if aw != 500 {
		t.Errorf("AdvanceWidth(0) = %d, want 500", aw)
	}

	// CFF-specific methods
	name := font.CFFFontName()
	if name != "TestOTTO" {
		t.Errorf("CFFFontName = %q, want %q", name, "TestOTTO")
	}

	gname := font.CFFGlyphName(0)
	if gname != ".notdef" {
		t.Errorf("GlyphName(0) = %q, want %q", gname, ".notdef")
	}
}

// --- CFF helper functions for test font generation ---

// buildTestCFFINDEX builds a CFF INDEX structure from a slice of byte slices.
func buildTestCFFINDEX(elements [][]byte) []byte {
	count := len(elements)
	if count == 0 {
		return []byte{0, 0}
	}

	// Calculate offsets (1-based)
	offsets := make([]uint32, count+1)
	offsets[0] = 1
	for i, elem := range elements {
		offsets[i+1] = offsets[i] + uint32(len(elem))
	}

	// Determine offSize
	maxOff := offsets[count]
	var offSize int
	if maxOff <= 0xFF {
		offSize = 1
	} else if maxOff <= 0xFFFF {
		offSize = 2
	} else if maxOff <= 0xFFFFFF {
		offSize = 3
	} else {
		offSize = 4
	}

	// Build INDEX
	size := 2 + 1 + (count+1)*offSize
	for _, elem := range elements {
		size += len(elem)
	}

	buf := make([]byte, 0, size)
	// count (uint16 big-endian)
	buf = append(buf, byte(count>>8), byte(count))
	// offSize
	buf = append(buf, byte(offSize))
	// offsets
	for _, off := range offsets {
		for j := offSize - 1; j >= 0; j-- {
			buf = append(buf, byte(off>>(j*8)))
		}
	}
	// data
	for _, elem := range elements {
		buf = append(buf, elem...)
	}

	return buf
}

// appendTestDictInt appends a DICT-encoded integer to buf.
func appendTestDictInt(buf []byte, val int32) []byte {
	if val >= -107 && val <= 108 {
		return append(buf, byte(val+139))
	}
	if val >= 108 && val <= 1131 {
		val -= 108
		return append(buf, byte((val>>8)+247), byte(val&0xFF))
	}
	if val >= -1131 && val <= -108 {
		val = -val - 108
		return append(buf, byte((val>>8)+251), byte(val&0xFF))
	}
	// Use 5-byte encoding (29 + int32)
	buf = append(buf, 29)
	buf = append(buf, byte(val>>24), byte(val>>16), byte(val>>8), byte(val))
	return buf
}

// generateMinimalOTTO creates a minimal valid OTTO font file for testing.
func generateMinimalOTTO(t *testing.T) []byte {
	t.Helper()

	// --- Build individual tables ---

	// head table (54 bytes)
	headData := make([]byte, 54)
	b := BinaryFrom(headData, false)
	var neg200 int16 = -200
	b.PutU16(1)
	b.PutU16(0)
	b.PutU16(1)
	b.PutU16(0)
	b.PutU32(0)          // checksumAdjustment
	b.PutU32(0x5F0F3CF5) // magicNumber
	b.PutU16(0x000B)     // flags
	b.PutU16(1000)       // unitsPerEm
	b.PutU64(0)
	b.PutU64(0)
	b.PutU16(0)
	b.PutU16(uint16(neg200))
	b.PutU16(600)
	b.PutU16(800)
	b.PutU16(0)
	b.PutU16(8)
	b.PutU16(2)
	b.PutU16(0)
	b.PutU16(0)

	// hhea table (36 bytes)
	hheaData := make([]byte, 36)
	b = BinaryFrom(hheaData, false)
	b.PutU32(0x00010000)
	b.PutU16(800)
	b.PutU16(uint16(neg200))
	for i := 0; i < 12; i++ {
		b.PutU16(0)
	}
	b.PutU16(600) // advanceWidthMax at correct position
	// Rewrite manually for clarity
	hheaData2 := make([]byte, 36)
	b = BinaryFrom(hheaData2, false)
	b.PutU32(0x00010000)     // version
	b.PutU16(800)            // ascent
	b.PutU16(uint16(neg200)) // descent
	b.PutU16(0)              // lineGap
	b.PutU16(600)            // advanceWidthMax
	b.PutU16(0)              // minLeftSideBearing
	b.PutU16(0)              // minRightSideBearing
	b.PutU16(800)            // xMaxExtent
	b.PutU16(1)              // caretSlopeRise
	b.PutU16(0)              // caretSlopeRun
	for i := 0; i < 5; i++ { // caretOffset + 4 reserved
		b.PutU16(0)
	}
	b.PutU16(0) // metricDataFormat
	b.PutU16(2) // numberOfHMetrics
	hheaData = hheaData2

	// maxp table (6 bytes for CFF)
	maxpData := make([]byte, 6)
	b = BinaryFrom(maxpData, false)
	b.PutU32(0x00005000)
	b.PutU16(2)

	// hmtx table (2 glyphs)
	hmtxData := make([]byte, 8)
	b = BinaryFrom(hmtxData, false)
	b.PutU16(500)
	b.PutU16(0)
	b.PutU16(500)
	b.PutU16(0)

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
	var stringStorage []byte
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
	b.PutU16(0)
	b.PutU16(uint16(len(records)))
	b.PutU16(6 + uint16(12*len(records)))
	for _, r := range records {
		b.PutU16(r.platformID)
		b.PutU16(r.encodingID)
		b.PutU16(r.languageID)
		b.PutU16(r.nameID)
		b.PutU16(r.length)
		b.PutU16(r.offset)
	}
	b.Append(stringStorage)

	// cmap table
	segCount := uint16(2)
	cmapSubtable := make([]byte, 14+int(segCount)*8+2)
	b = BinaryFrom(cmapSubtable, false)
	b.PutU16(4)
	b.PutU16(0)
	b.PutU16(0)
	b.PutU16(segCount * 2)
	sr, es, rs := calcSearchParams(int(segCount))
	b.PutU16(sr)
	b.PutU16(es)
	b.PutU16(rs)
	b.PutU16(0x0041)
	b.PutU16(0xFFFF)
	b.PutU16(0)
	b.PutU16(0x0041)
	b.PutU16(0xFFFF)
	var delta int16 = 1 - 0x0041
	b.PutU16(uint16(delta))
	b.PutU16(1)
	b.PutU16(0)
	b.PutU16(0)
	b2 := BinaryFrom(cmapSubtable[2:4], false)
	b2.PutU16(uint16(len(cmapSubtable)))

	cmapData := make([]byte, 4+8+len(cmapSubtable))
	b = BinaryFrom(cmapData, false)
	b.PutU16(0)
	b.PutU16(1)
	b.PutU16(3)
	b.PutU16(1)
	b.PutU32(12)
	b.Append(cmapSubtable)

	// OS/2 table (86 bytes for version 1)
	os2Data := make([]byte, 86)
	b = BinaryFrom(os2Data, false)
	b.PutU16(1)
	b.PutU16(500)
	b.PutU16(400)
	b.PutU16(5)
	b.PutU16(0)
	for i := 0; i < 10; i++ {
		b.PutU16(0)
	}
	b.Append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) // panose
	b.Append([]byte{0, 0, 0, 0})
	b.Append([]byte{0, 0, 0, 0})
	b.Append([]byte{0, 0, 0, 0})
	b.Append([]byte{0, 0, 0, 4})
	b.Append([]byte("    "))
	b.PutU16(0x0040)
	b.PutU16(0x0041)
	b.PutU16(0x0041)
	b.PutU16(800)
	b.PutU16(uint16(neg200))
	b.PutU16(0)
	b.PutU16(800)
	b.PutU16(200)

	// --- Build CFF table ---
	// Layout:
	//   [0-3]  Header (4 bytes)
	//   [4-16] Name INDEX (13 bytes: count=1, offSize=1, offsets=[1,9], "TestOTTO")
	//   [17-?] Top DICT INDEX
	//   [?]    String INDEX (2 bytes, empty)
	//   [?]    Global Subr INDEX (2 bytes, empty)
	//   [?]    Charset (3 bytes: format 0, SID 1)
	//   [?]    CharStrings INDEX

	cs0 := []byte{14} // endchar
	cs1 := []byte{14} // endchar

	charsetData := []byte{0, 0, 1} // format 0, SID=1 ("space")
	strIdx := []byte{0, 0}
	gsubrIdx := []byte{0, 0}

	nameIdxBytes := []byte{0, 1, 1, 1, 9, 'T', 'e', 's', 't', 'O', 'T', 'T', 'O'}

	// Iteratively compute correct offsets
	var cffData []byte
	// Start with overestimate: use 3 bytes per DICT integer (28, b1, b2)
	// Each entry: 3 bytes value + 1 byte operator = 4 bytes. 2 entries = 8 bytes top dict data.
	// Top DICT INDEX overhead: 2(count) + 1(offSize) + 2(offsets) = 5 + data
	estTopDictSize := 5 + 8
	estCharsetOff := 4 + len(nameIdxBytes) + estTopDictSize + len(strIdx) + len(gsubrIdx)

	for pass := 0; pass < 5; pass++ {
		charsetOff := estCharsetOff
		charstringsOff := charsetOff + len(charsetData)

		topDictData := make([]byte, 0, 16)
		topDictData = appendTestDictInt(topDictData, int32(charsetOff))
		topDictData = append(topDictData, 15)
		topDictData = appendTestDictInt(topDictData, int32(charstringsOff))
		topDictData = append(topDictData, 17)

		topDictIdx := buildTestCFFINDEX([][]byte{topDictData})
		newCharsetOff := 4 + len(nameIdxBytes) + len(topDictIdx) + len(strIdx) + len(gsubrIdx)

		if newCharsetOff == estCharsetOff {
			csIdx := buildTestCFFINDEX([][]byte{cs0, cs1})
			buf := make([]byte, 0, 256)
			buf = append(buf, 1, 0, 4, 1)
			buf = append(buf, nameIdxBytes...)
			buf = append(buf, topDictIdx...)
			buf = append(buf, strIdx...)
			buf = append(buf, gsubrIdx...)
			buf = append(buf, charsetData...)
			buf = append(buf, csIdx...)
			cffData = buf
			break
		}
		estCharsetOff = newCharsetOff
	}

	// post table (format 3.0 - 32 bytes)
	postData := make([]byte, 32)
	b = BinaryFrom(postData, false)
	var neg100 int16 = -100
	b.PutU32(0x00030000)
	b.PutU16(0)
	b.PutU16(0)
	b.PutU16(uint16(neg100))
	b.PutU16(50)
	b.PutU32(0)
	b.PutU32(0)
	b.PutU32(0)
	b.PutU32(0)
	b.PutU32(0)

	// --- Assemble font file ---
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
	fileHeaderSize := 12 + 16*numTables
	offset := uint32(fileHeaderSize)

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
