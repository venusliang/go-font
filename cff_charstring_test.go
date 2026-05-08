package gofont

import (
	"testing"
)

func TestDecodeCharStringEndChar(t *testing.T) {
	// Simple endchar-only charstring (empty glyph like space)
	data := []byte{14} // endchar
	globalSubrs := &CFFINDEX{}
	localSubrs := &CFFINDEX{}

	outline, err := decodeCharString(data, globalSubrs, localSubrs, 0, 0)
	if err != nil {
		t.Fatalf("decodeCharString error: %v", err)
	}
	if outline.NumSegments() != 0 {
		t.Errorf("NumSegments = %d, want 0", outline.NumSegments())
	}
}

func TestDecodeCharStringLine(t *testing.T) {
	// Use values in range -107 to 108 for single-byte encoding
	// rmoveto(50, 60) + rlineto(70, 80) + endchar
	data := []byte{
		50 + 139, // 50
		60 + 139, // 60
		21,       // rmoveto
		70 + 139, // 70
		80 + 139, // 80
		5,        // rlineto
		14,       // endchar
	}

	outline, err := decodeCharString(data, &CFFINDEX{}, &CFFINDEX{}, 0, 0)
	if err != nil {
		t.Fatalf("decodeCharString error: %v", err)
	}

	if outline.NumSegments() != 2 {
		t.Fatalf("NumSegments = %d, want 2", outline.NumSegments())
	}

	// First segment: moveto
	seg := outline.Segments()[0]
	if seg.Op != CFFOpMoveTo {
		t.Errorf("segment 0 op = %v, want MoveTo", seg.Op)
	}
	if seg.Args[0] != 50 || seg.Args[1] != 60 {
		t.Errorf("segment 0 args = %v, want [50,60,...]", seg.Args)
	}

	// Second segment: lineto
	seg = outline.Segments()[1]
	if seg.Op != CFFOpLineTo {
		t.Errorf("segment 1 op = %v, want LineTo", seg.Op)
	}
	if seg.Args[0] != 70 || seg.Args[1] != 80 {
		t.Errorf("segment 1 args = %v, want [70,80,...]", seg.Args)
	}
}

func TestDecodeCharStringCurve(t *testing.T) {
	// rmoveto(10, 20) + rrcurveto(30,40,50,60,70,80) + endchar
	data := []byte{
		10 + 139, // 10
		20 + 139, // 20
		21,       // rmoveto
		30 + 139, // 30
		40 + 139, // 40
		50 + 139, // 50
		60 + 139, // 60
		70 + 139, // 70
		80 + 139, // 80
		8,        // rrcurveto
		14,       // endchar
	}

	outline, err := decodeCharString(data, &CFFINDEX{}, &CFFINDEX{}, 0, 0)
	if err != nil {
		t.Fatalf("decodeCharString error: %v", err)
	}

	if outline.NumSegments() != 2 {
		t.Fatalf("NumSegments = %d, want 2", outline.NumSegments())
	}

	seg := outline.Segments()[1]
	if seg.Op != CFFOpCurveTo {
		t.Errorf("segment 1 op = %v, want CurveTo", seg.Op)
	}
	if seg.Args[0] != 30 || seg.Args[1] != 40 || seg.Args[2] != 50 || seg.Args[3] != 60 || seg.Args[4] != 70 || seg.Args[5] != 80 {
		t.Errorf("segment 1 args = %v, want [30,40,50,60,70,80]", seg.Args)
	}
}

func TestDecodeCharStringHmoveto(t *testing.T) {
	// hmoveto(100) + endchar
	data := []byte{
		100 + 139, // 100
		22,        // hmoveto
		14,        // endchar
	}

	outline, err := decodeCharString(data, &CFFINDEX{}, &CFFINDEX{}, 0, 0)
	if err != nil {
		t.Fatalf("decodeCharString error: %v", err)
	}

	if outline.NumSegments() != 1 {
		t.Fatalf("NumSegments = %d, want 1", outline.NumSegments())
	}

	seg := outline.Segments()[0]
	if seg.Op != CFFOpMoveTo {
		t.Errorf("op = %v, want MoveTo", seg.Op)
	}
	if seg.Args[0] != 100 || seg.Args[1] != 0 {
		t.Errorf("args = %v, want [100,0,...]", seg.Args)
	}
}

func TestDecodeCharStringWithSubr(t *testing.T) {
	// Local subr 0: rlineto(10, 20)
	subrData := []byte{
		10 + 139, // 10
		20 + 139, // 20
		5,        // rlineto
		11,       // return
	}

	localSubrs := buildTestCFFINDEXFromBytes([][]byte{subrData})
	globalSubrs := &CFFINDEX{count: 0}

	// Main: rmoveto(50, 50) + callsubr(0) + endchar
	// subr bias for count < 1240 = 107
	// So we push (0 - 107) = -107, which encodes as byte 32 (32 = -107 + 139)
	data := []byte{
		50 + 139, // 50
		50 + 139, // 50
		21,       // rmoveto
		32,       // -107 (= 0 - bias 107)
		10,       // callsubr
		14,       // endchar
	}

	outline, err := decodeCharString(data, globalSubrs, localSubrs, 0, 0)
	if err != nil {
		t.Fatalf("decodeCharString error: %v", err)
	}

	if outline.NumSegments() != 2 {
		t.Fatalf("NumSegments = %d, want 2", outline.NumSegments())
	}

	// Second segment should be the lineto from the subroutine
	seg := outline.Segments()[1]
	if seg.Op != CFFOpLineTo {
		t.Errorf("segment 1 op = %v, want LineTo", seg.Op)
	}
	if seg.Args[0] != 10 || seg.Args[1] != 20 {
		t.Errorf("segment 1 args = %v, want [10,20,...]", seg.Args)
	}
}

func TestCFFAllGlyphOutlines(t *testing.T) {
	fontData := loadOTTOFont(t)
	font, err := Parse(fontData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if font.cff == nil {
		t.Fatal("CFF should be parsed")
	}

	outlines, err := font.cff.DecodeOutlines()
	if err != nil {
		t.Fatalf("DecodeOutlines error: %v", err)
	}

	if len(outlines) != 2 {
		t.Fatalf("len(outlines) = %d, want 2", len(outlines))
	}

	// Both glyphs are just endchar (empty)
	for i, o := range outlines {
		if o == nil {
			t.Errorf("outline[%d] is nil", i)
		}
	}
}

func TestCFFOutlineAtAPI(t *testing.T) {
	fontData := loadOTTOFont(t)
	font, err := Parse(fontData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// CFFOutlineAt should work
	o := font.CFFOutlineAt(0)
	if o == nil {
		t.Error("CFFOutlineAt(0) should not be nil")
	}

	// Out of range
	o = font.CFFOutlineAt(99)
	if o != nil {
		t.Error("CFFOutlineAt(99) should be nil")
	}

	// CFFOutlineForRune for 'A' (mapped to glyph 1)
	o = font.CFFOutlineForRune('A')
	if o == nil {
		t.Error("CFFOutlineForRune('A') should not be nil")
	}

	// Unmapped rune
	o = font.CFFOutlineForRune('Z')
	if o != nil {
		t.Error("CFFOutlineForRune('Z') should be nil (not mapped)")
	}
}

// buildTestCFFINDEXFromBytes wraps buildTestCFFINDEX to return a *CFFINDEX.
func buildTestCFFINDEXFromBytes(elements [][]byte) *CFFINDEX {
	raw := buildTestCFFINDEX(elements)
	idx, _, err := parseCFFINDEX(raw, 0)
	if err != nil {
		return &CFFINDEX{count: uint16(len(elements))}
	}
	return &idx
}
