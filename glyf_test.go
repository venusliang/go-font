package gofonts

import (
	"testing"
)

func getGlyphs(t *testing.T) []*Glyph {
	t.Helper()
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.glyf == nil {
		t.Fatal("glyf data is nil")
	}
	return ttf.glyf
}

func TestParseGlyf(t *testing.T) {
	glyphs := getGlyphs(t)

	if len(glyphs) != 43 {
		t.Fatalf("glyph count: got %d, want 43", len(glyphs))
	}

	// glyph[0]: 2 contours, 8 points
	g0 := glyphs[0]
	if g0.header.numberOfContours != 2 {
		t.Errorf("glyph[0] numberOfContours: got %d, want 2", g0.header.numberOfContours)
	}
	if g0.simpleGlyph == nil {
		t.Fatal("glyph[0] is not a simple glyph")
	}
	if g0.header.xMin != 50 || g0.header.yMin != -112 {
		t.Errorf("glyph[0] bbox min: got (%d,%d), want (50,-112)", g0.header.xMin, g0.header.yMin)
	}
	if g0.header.xMax != 379 || g0.header.yMax != 712 {
		t.Errorf("glyph[0] bbox max: got (%d,%d), want (379,712)", g0.header.xMax, g0.header.yMax)
	}
	if len(g0.simpleGlyph.endPtsOfContours) != 2 {
		t.Errorf("glyph[0] contour count: got %d, want 2", len(g0.simpleGlyph.endPtsOfContours))
	}
	if g0.simpleGlyph.endPtsOfContours[0] != 3 || g0.simpleGlyph.endPtsOfContours[1] != 7 {
		t.Errorf("glyph[0] endPtsOfContours: got %v, want [3, 7]", g0.simpleGlyph.endPtsOfContours)
	}
	if len(g0.simpleGlyph.xCoordinates) != 8 {
		t.Errorf("glyph[0] point count: got %d, want 8", len(g0.simpleGlyph.xCoordinates))
	}
	if len(g0.simpleGlyph.yCoordinates) != 8 {
		t.Errorf("glyph[0] yCoordinates count: got %d, want 8", len(g0.simpleGlyph.yCoordinates))
	}

	// glyph[1]: 1 contour, 78 points
	g1 := glyphs[1]
	if g1.header.numberOfContours != 1 {
		t.Errorf("glyph[1] numberOfContours: got %d, want 1", g1.header.numberOfContours)
	}
	if len(g1.simpleGlyph.xCoordinates) != 78 {
		t.Errorf("glyph[1] point count: got %d, want 78", len(g1.simpleGlyph.xCoordinates))
	}
}

func TestRoundTripGlyf(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Write glyf, then re-parse and compare
	glyfWritten, locaOffsets := writeGlyf(ttf.glyf)
	newLoca := &Loca{offsets: locaOffsets}

	glyphs2, err := parseGlyf(glyfWritten, newLoca)
	if err != nil {
		t.Fatal(err)
	}

	if len(glyphs2) != len(ttf.glyf) {
		t.Fatalf("glyph count mismatch: %d vs %d", len(glyphs2), len(ttf.glyf))
	}

	for i, orig := range ttf.glyf {
		g2 := glyphs2[i]
		if orig.header != g2.header {
			t.Errorf("glyph[%d] header mismatch: %+v vs %+v", i, g2.header, orig.header)
		}
		if orig.simpleGlyph != nil && g2.simpleGlyph != nil {
			sg := orig.simpleGlyph
			sg2 := g2.simpleGlyph
			if len(sg2.endPtsOfContours) != len(sg.endPtsOfContours) {
				t.Errorf("glyph[%d] contour count mismatch", i)
				continue
			}
			for j, ep := range sg.endPtsOfContours {
				if sg2.endPtsOfContours[j] != ep {
					t.Errorf("glyph[%d] endPtsOfContours[%d] mismatch", i, j)
				}
			}
			if len(sg2.xCoordinates) != len(sg.xCoordinates) {
				t.Errorf("glyph[%d] xCoordinates count mismatch: %d vs %d", i, len(sg2.xCoordinates), len(sg.xCoordinates))
				continue
			}
			for j, x := range sg.xCoordinates {
				if sg2.xCoordinates[j] != x {
					t.Errorf("glyph[%d] x[%d]: got %d, want %d", i, j, sg2.xCoordinates[j], x)
				}
			}
			for j, y := range sg.yCoordinates {
				if sg2.yCoordinates[j] != y {
					t.Errorf("glyph[%d] y[%d]: got %d, want %d", i, j, sg2.yCoordinates[j], y)
				}
			}
		}
	}
}
