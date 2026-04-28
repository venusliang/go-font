package gofont

import (
	"testing"
)

// --- P1 Tests ---

func TestUnitsPerEm(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if v := ttf.UnitsPerEm(); v != 1024 {
		t.Errorf("UnitsPerEm: got %d, want 1024", v)
	}
}

func TestFontBBox(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	xMin, yMin, xMax, yMax := ttf.FontBBox()
	// Test font has known bbox values
	if xMin == 0 && yMin == 0 && xMax == 0 && yMax == 0 {
		t.Error("FontBBox returned all zeros")
	}
}

func TestAscentDescent(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	a := ttf.Ascent()
	d := ttf.Descent()
	if a == 0 {
		t.Error("Ascent should not be 0")
	}
	if d == 0 {
		t.Error("Descent should not be 0")
	}
	if a <= 0 {
		t.Errorf("Ascent should be positive, got %d", a)
	}
	if d >= 0 {
		t.Errorf("Descent should be negative, got %d", d)
	}
}

func TestAdvanceWidth(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	w := ttf.AdvanceWidth(0)
	if w == 0 {
		t.Error("AdvanceWidth(0) should not be 0")
	}

	// Out of range
	if w := ttf.AdvanceWidth(100); w != 0 {
		t.Errorf("AdvanceWidth(100): got %d, want 0", w)
	}
}

func TestLeftSideBearing(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	lsb := ttf.LeftSideBearing(0)
	_ = lsb // just verify it doesn't crash

	// Out of range
	if lsb := ttf.LeftSideBearing(100); lsb != 0 {
		t.Errorf("LeftSideBearing(100): got %d, want 0", lsb)
	}
}

func TestAdvanceWidthForRune(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	// 0xE001 maps to glyph 1
	w := ttf.AdvanceWidthForRune(0xE001)
	w1 := ttf.AdvanceWidth(1)
	if w != w1 {
		t.Errorf("AdvanceWidthForRune(0xE001)=%d != AdvanceWidth(1)=%d", w, w1)
	}

	// Unmapped rune
	if w := ttf.AdvanceWidthForRune(0x41); w != 0 {
		t.Errorf("AdvanceWidthForRune('A'): got %d, want 0", w)
	}
}

func TestIsSimpleGlyph(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	// glyph 0 is a simple glyph (2 contours)
	if !ttf.IsSimpleGlyph(0) {
		t.Error("glyph 0 should be simple")
	}
	if ttf.IsCompositeGlyph(0) {
		t.Error("glyph 0 should not be composite")
	}
	// Out of range
	if ttf.IsSimpleGlyph(-1) {
		t.Error("index -1 should not be simple")
	}
	if ttf.IsSimpleGlyph(100) {
		t.Error("index 100 should not be simple")
	}
}

func TestGlyphBBox(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	xMin, yMin, xMax, yMax, ok := ttf.GlyphBBox(0)
	if !ok {
		t.Fatal("GlyphBBox(0) ok=false")
	}
	if xMin != 50 || yMin != -112 || xMax != 379 || yMax != 712 {
		t.Errorf("GlyphBBox(0): got (%d,%d,%d,%d), want (50,-112,379,712)", xMin, yMin, xMax, yMax)
	}

	_, _, _, _, ok = ttf.GlyphBBox(100)
	if ok {
		t.Error("GlyphBBox(100) should return ok=false")
	}
}

func TestPointCount(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	// glyph 0 has 8 points
	if n := ttf.PointCount(0); n != 8 {
		t.Errorf("PointCount(0): got %d, want 8", n)
	}
	// glyph 1 has 78 points
	if n := ttf.PointCount(1); n != 78 {
		t.Errorf("PointCount(1): got %d, want 78", n)
	}
	if n := ttf.PointCount(100); n != 0 {
		t.Errorf("PointCount(100): got %d, want 0", n)
	}
}

func TestContourCount(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if n := ttf.ContourCount(0); n != 2 {
		t.Errorf("ContourCount(0): got %d, want 2", n)
	}
	if n := ttf.ContourCount(1); n != 1 {
		t.Errorf("ContourCount(1): got %d, want 1", n)
	}
	if n := ttf.ContourCount(100); n != 0 {
		t.Errorf("ContourCount(100): got %d, want 0", n)
	}
}

func TestFontFamilyAndFullName(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	family := ttf.FontFamily()
	if family == "" {
		t.Error("FontFamily returned empty string")
	} else {
		t.Logf("FontFamily: %q", family)
	}
	full := ttf.FontFullName()
	if full == "" {
		t.Error("FontFullName returned empty string")
	} else {
		t.Logf("FontFullName: %q", full)
	}
}

// --- P2 Tests ---

func TestSetAdvanceWidth(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	origWidth := ttf.AdvanceWidth(1)
	err = ttf.SetAdvanceWidth(1, origWidth+100)
	if err != nil {
		t.Fatal(err)
	}
	if w := ttf.AdvanceWidth(1); w != origWidth+100 {
		t.Errorf("AdvanceWidth after set: got %d, want %d", w, origWidth+100)
	}

	// Out of range
	err = ttf.SetAdvanceWidth(100, 500)
	if err == nil {
		t.Error("expected error for out-of-range glyph ID")
	}
}

func TestSetLeftSideBearing(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	err = ttf.SetLeftSideBearing(1, 42)
	if err != nil {
		t.Fatal(err)
	}
	if lsb := ttf.LeftSideBearing(1); lsb != 42 {
		t.Errorf("LeftSideBearing after set: got %d, want 42", lsb)
	}
}

func TestSetFontName(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	ttf.SetFontFamily("TestFont")
	if v := ttf.FontFamily(); v != "TestFont" {
		t.Errorf("FontFamily after set: got %q, want %q", v, "TestFont")
	}

	ttf.SetFontFullName("TestFont Bold")
	if v := ttf.FontFullName(); v != "TestFont Bold" {
		t.Errorf("FontFullName after set: got %q, want %q", v, "TestFont Bold")
	}
}

func TestSetFontNameSerialize(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	ttf.SetFontFamily("RenamedFont")
	ttf.SetFontFullName("RenamedFont Regular")

	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if v := ttf2.FontFamily(); v != "RenamedFont" {
		t.Errorf("FontFamily after round-trip: got %q, want %q", v, "RenamedFont")
	}
	if v := ttf2.FontFullName(); v != "RenamedFont Regular" {
		t.Errorf("FontFullName after round-trip: got %q, want %q", v, "RenamedFont Regular")
	}
}

func TestTranslateGlyph(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	origXMin, origYMin, _, _, _ := ttf.GlyphBBox(0)

	err = ttf.TranslateGlyph(0, 100, -50)
	if err != nil {
		t.Fatal(err)
	}

	xMin, yMin, _, _, _ := ttf.GlyphBBox(0)
	if xMin != origXMin+100 {
		t.Errorf("xMin after translate: got %d, want %d", xMin, origXMin+100)
	}
	if yMin != origYMin-50 {
		t.Errorf("yMin after translate: got %d, want %d", yMin, origYMin-50)
	}

	// Verify coordinates shifted
	g := ttf.GlyphAt(0)
	if g.simpleGlyph.xCoordinates[0] != 150 { // original 50 + 100
		t.Errorf("x[0] after translate: got %d, want 150", g.simpleGlyph.xCoordinates[0])
	}
}

func TestScaleGlyph(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	err = ttf.ScaleGlyph(0, 2.0, 2.0)
	if err != nil {
		t.Fatal(err)
	}

	xMin, _, xMax, _, _ := ttf.GlyphBBox(0)
	// Original was (50, -112, 379, 712) — scaled by 2x
	if xMin != 100 {
		t.Errorf("xMin after scale: got %d, want 100", xMin)
	}
	if xMax != 758 {
		t.Errorf("xMax after scale: got %d, want 758", xMax)
	}

	// Out of range
	err = ttf.ScaleGlyph(100, 1.0, 1.0)
	if err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestAppendGlyph(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	origCount := ttf.NumGlyphs()

	// Copy glyph 0 and append as new glyph
	src := ttf.GlyphAt(0)
	newG := &Glyph{header: src.header}
	if src.simpleGlyph != nil {
		sg := *src.simpleGlyph
		newG.simpleGlyph = &sg
	}

	idx, err := ttf.AppendGlyph(newG)
	if err != nil {
		t.Fatal(err)
	}
	if idx != origCount {
		t.Errorf("AppendGlyph index: got %d, want %d", idx, origCount)
	}
	if n := ttf.NumGlyphs(); n != origCount+1 {
		t.Errorf("NumGlyphs after append: got %d, want %d", n, origCount+1)
	}
}

func TestAppendGlyphSerialize(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	src := ttf.GlyphAt(0)
	newG := &Glyph{header: src.header}
	if src.simpleGlyph != nil {
		sg := *src.simpleGlyph
		newG.simpleGlyph = &sg
	}

	ttf.AppendGlyph(newG)
	ttf.SetRuneMapping(0x41, uint16(ttf.NumGlyphs()-1))

	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if n := ttf2.NumGlyphs(); n != ttf.NumGlyphs() {
		t.Errorf("NumGlyphs after round-trip: got %d, want %d", n, ttf.NumGlyphs())
	}
}

// --- P3 Tests ---

func TestSubset(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	origCount := ttf.NumGlyphs()

	// Keep only a few runes: glyph 0 (.notdef) + glyphs for E001, E002
	err = ttf.Subset([]rune{0xE001, 0xE002})
	if err != nil {
		t.Fatal(err)
	}

	// Should have glyph 0 + glyph for E001 + glyph for E002 = 3
	if n := ttf.NumGlyphs(); n != 3 {
		t.Errorf("NumGlyphs after subset: got %d, want 3 (had %d before)", n, origCount)
	}

	// Verify mappings still work
	if gid := ttf.RuneToGlyphID(0xE001); gid == 0 {
		t.Error("0xE001 should still be mapped")
	}
	if gid := ttf.RuneToGlyphID(0xE030); gid != 0 {
		t.Errorf("0xE030 should be unmapped after subset, got %d", gid)
	}
}

func TestSubsetSerialize(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	err = ttf.Subset([]rune{0xE001, 0xE002, 0xE003})
	if err != nil {
		t.Fatal(err)
	}

	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse subset font: %v", err)
	}

	if n := ttf2.NumGlyphs(); n != 4 { // .notdef + 3
		t.Errorf("NumGlyphs after round-trip: got %d, want 4", n)
	}

	// Verify cmap works
	if gid := ttf2.RuneToGlyphID(0xE001); gid == 0 {
		t.Error("0xE001 should be mapped after round-trip")
	}
}

func TestCopyGlyph(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	oxMin, oyMin, oxMax, oyMax, _ := ttf.GlyphBBox(0)
	err = ttf.CopyGlyph(0, 1)
	if err != nil {
		t.Fatal(err)
	}

	nxMin, nyMin, nxMax, nyMax, _ := ttf.GlyphBBox(1)
	if oxMin != nxMin || oyMin != nyMin || oxMax != nxMax || oyMax != nyMax {
		t.Errorf("CopyGlyph bbox mismatch: src=(%d,%d,%d,%d) dst=(%d,%d,%d,%d)",
			oxMin, oyMin, oxMax, oyMax, nxMin, nyMin, nxMax, nyMax)
	}
}

func TestSetRuneMappings(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	mappings := map[rune]uint16{
		'A': 1,
		'B': 2,
		'C': 3,
	}
	err = ttf.SetRuneMappings(mappings)
	if err != nil {
		t.Fatal(err)
	}

	if gid := ttf.RuneToGlyphID('A'); gid != 1 {
		t.Errorf("RuneToGlyphID('A'): got %d, want 1", gid)
	}
	if gid := ttf.RuneToGlyphID('B'); gid != 2 {
		t.Errorf("RuneToGlyphID('B'): got %d, want 2", gid)
	}

	// Out of range should fail
	err = ttf.SetRuneMappings(map[rune]uint16{'D': 200})
	if err == nil {
		t.Error("expected error for out-of-range glyph ID")
	}
}

func TestMappedRunes(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	runes := ttf.MappedRunes()
	if len(runes) == 0 {
		t.Fatal("MappedRunes returned empty")
	}

	// Should be sorted
	for i := 1; i < len(runes); i++ {
		if runes[i] <= runes[i-1] {
			t.Errorf("MappedRunes not sorted: [%d]=%04X <= [%d]=%04X", i-1, runes[i-1], i, runes[i])
		}
	}
	t.Logf("MappedRunes: %d entries, first=U+%04X, last=U+%04X", len(runes), runes[0], runes[len(runes)-1])
}

func TestSetAdvanceWidthSerialize(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	ttf.SetAdvanceWidth(1, 600)

	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatal(err)
	}
	if w := ttf2.AdvanceWidth(1); w != 600 {
		t.Errorf("AdvanceWidth after round-trip: got %d, want 600", w)
	}
}
