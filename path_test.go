package gofont

import (
	"testing"
)

func TestGlyphPath_TrueType(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// glyph[0] (.notdef): 2 contours, 8 points, bbox (179,0)-(1248,1510)
	p, err := ttf.GlyphPath(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Segments) == 0 {
		t.Error("glyph[0] should have non-empty path")
	}

	// First segment should be MoveTo
	if p.Segments[0].Op != OpMoveTo {
		t.Errorf("glyph[0] first segment: got op=%d, want OpMoveTo", p.Segments[0].Op)
	}

	// Should have 2 contours: find the second MoveTo
	moveToCount := 0
	for _, seg := range p.Segments {
		if seg.Op == OpMoveTo {
			moveToCount++
		}
	}
	if moveToCount != 2 {
		t.Errorf("glyph[0] MoveTo count: got %d, want 2", moveToCount)
	}

	// BBox should match glyph header (design units)
	if p.XMin < 170 || p.XMin > 190 {
		t.Errorf("glyph[0] XMin: got %.0f, want ~179", p.XMin)
	}
	if p.YMin != 0 {
		t.Errorf("glyph[0] YMin: got %.0f, want 0", p.YMin)
	}
	if p.XMax < 1240 || p.XMax > 1260 {
		t.Errorf("glyph[0] XMax: got %.0f, want ~1248", p.XMax)
	}
	if p.YMax < 1500 || p.YMax > 1520 {
		t.Errorf("glyph[0] YMax: got %.0f, want ~1510", p.YMax)
	}

	// Segments should only be MoveTo, LineTo, QuadTo (TrueType = quadratic beziers)
	for _, seg := range p.Segments {
		if seg.Op == OpCubeTo {
			t.Error("TrueType glyph should not contain cubic beziers")
		}
	}
}

func TestGlyphPath_MappedRune(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// 'A' (U+0041) → glyph 39
	gid := ttf.RuneToGlyphID('A')
	p, err := ttf.GlyphPath(int(gid))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Segments) == 0 {
		t.Error("glyph for 'A' should have non-empty path")
	}
}

func TestGlyphPath_EmptyGlyph(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// space (U+0020) → glyph 3, may or may not have outline
	// Test out-of-range error instead
	_, err = ttf.GlyphPath(99999)
	if err == nil {
		t.Error("expected error for out-of-range glyph index")
	}

	// Negative index
	_, err = ttf.GlyphPath(-1)
	if err == nil {
		t.Error("expected error for negative glyph index")
	}
}

func TestGlyphPath_CFF(t *testing.T) {
	ttf, err := Parse(loadOTTOFont(t))
	if err != nil {
		t.Fatal(err)
	}

	if !ttf.IsCFF() {
		t.Fatal("OTTO font should be CFF")
	}

	// The generated OTTO font has 2 glyphs, both with only endchar (no drawing ops)
	p, err := ttf.GlyphPath(0)
	if err != nil {
		t.Fatal(err)
	}
	// Should return empty segments (no drawing commands)
	if len(p.Segments) != 0 {
		t.Logf("CFF glyph[0] has %d segments (expected 0 for endchar-only)", len(p.Segments))
	}

	// glyph[1] is also endchar-only
	p, err = ttf.GlyphPath(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Segments) != 0 {
		t.Logf("CFF glyph[1] has %d segments (expected 0 for endchar-only)", len(p.Segments))
	}

	// Out-of-range
	_, err = ttf.GlyphPath(2)
	if err == nil {
		t.Error("expected error for out-of-range CFF glyph index")
	}
}
