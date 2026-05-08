package gofont

import (
	"testing"
)

func TestParseCFF(t *testing.T) {
	data := loadOTTOFont(t)
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if font.cff == nil {
		t.Fatal("CFF table should be parsed")
	}

	cff := font.cff

	// Header
	if cff.header.majorVersion != 1 {
		t.Errorf("majorVersion = %d, want 1", cff.header.majorVersion)
	}
	if cff.header.minorVersion != 0 {
		t.Errorf("minorVersion = %d, want 0", cff.header.minorVersion)
	}

	// Font name
	name := cff.FontName()
	if name != "TestOTTO" {
		t.Errorf("FontName = %q, want %q", name, "TestOTTO")
	}

	// NumGlyphs from CharStrings INDEX
	if cff.NumGlyphs() != 2 {
		t.Errorf("NumGlyphs = %d, want 2", cff.NumGlyphs())
	}
}

func TestCFFGlyphNames(t *testing.T) {
	data := loadOTTOFont(t)
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	cff := font.cff

	// Glyph 0 is always .notdef
	name := cff.GlyphName(0)
	if name != ".notdef" {
		t.Errorf("GlyphName(0) = %q, want %q", name, ".notdef")
	}

	// Glyph 1 = SID 1 = "space"
	name = cff.GlyphName(1)
	if name != "space" {
		t.Errorf("GlyphName(1) = %q, want %q", name, "space")
	}
}

func TestCFFCharStringsIndex(t *testing.T) {
	data := loadOTTOFont(t)
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	cff := font.cff

	// CharStrings INDEX should have 2 entries
	if cff.charStrings.count != 2 {
		t.Errorf("charStrings.count = %d, want 2", cff.charStrings.count)
	}

	// Each charstring should be accessible
	for i := 0; i < 2; i++ {
		csData, err := cff.CharStringData(i)
		if err != nil {
			t.Errorf("CharStringData(%d) error: %v", i, err)
		}
		if len(csData) == 0 {
			t.Errorf("CharStringData(%d) should not be empty", i)
		}
	}

	// Out of range
	_, err = cff.CharStringData(2)
	if err == nil {
		t.Error("CharStringData(2) should return error for out of range")
	}
}

func TestCFFPrivateDict(t *testing.T) {
	data := loadOTTOFont(t)
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	cff := font.cff

	// Our test font doesn't have a Private DICT (no Private operator in Top DICT)
	// So privateSize should be 0
	if cff.topDict.privateSize != 0 {
		t.Errorf("privateSize = %d, want 0 (no Private DICT in test font)", cff.topDict.privateSize)
	}
}

func TestCFFSIDResolution(t *testing.T) {
	// Test standard string resolution
	tests := []struct {
		sid  uint16
		want string
	}{
		{0, ".notdef"},
		{1, "space"},
		{34, "A"},
		{36, "C"},
	}

	for _, tt := range tests {
		got := cffSIDToString(tt.sid, &CFFINDEX{})
		if got != tt.want {
			t.Errorf("cffSIDToString(%d) = %q, want %q", tt.sid, got, tt.want)
		}
	}

	// Test custom string (SID >= len(cffStandardStrings))
	customStrings := buildTestCFFINDEX([][]byte{[]byte("custom1"), []byte("custom2")})
	idx, _, err := parseCFFINDEX(customStrings, 0)
	if err != nil {
		t.Fatalf("parseCFFINDEX error: %v", err)
	}

	baseSID := uint16(len(cffStandardStrings))
	got := cffSIDToString(baseSID, &idx)
	if got != "custom1" {
		t.Errorf("cffSIDToString(%d) = %q, want %q", baseSID, got, "custom1")
	}
	got = cffSIDToString(baseSID+1, &idx)
	if got != "custom2" {
		t.Errorf("cffSIDToString(%d) = %q, want %q", baseSID+1, got, "custom2")
	}
}

func TestRoundTripCFFRaw(t *testing.T) {
	data := loadOTTOFont(t)
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Store original CFF raw data
	originalCFF := font.rawTables["CFF "]

	serialized, err := font.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	font2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("Re-parse failed: %v", err)
	}

	// CFF raw bytes should be identical after round-trip
	roundTripCFF := font2.rawTables["CFF "]
	if len(roundTripCFF) != len(originalCFF) {
		t.Errorf("CFF size mismatch: got %d, want %d", len(roundTripCFF), len(originalCFF))
	}
	for i := range originalCFF {
		if roundTripCFF[i] != originalCFF[i] {
			t.Errorf("CFF byte mismatch at offset %d", i)
			break
		}
	}

	// CFF parsed data should match
	if font2.cff == nil {
		t.Fatal("CFF should be re-parsed after round-trip")
	}
	if font2.cff.NumGlyphs() != font.cff.NumGlyphs() {
		t.Errorf("NumGlyphs mismatch after round-trip: got %d, want %d", font2.cff.NumGlyphs(), font.cff.NumGlyphs())
	}
}
