package gofont

import (
	"testing"
)

func TestTTFToEOT(t *testing.T) {
	// TTF → Parse → SerializeEOT → ParseEOT → compare
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	eotData, err := ttf.SerializeEOT()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := ParseEOT(eotData)
	if err != nil {
		t.Fatal(err)
	}

	if ttf2.head.unitsPerEm != ttf.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: %d vs %d", ttf2.head.unitsPerEm, ttf.head.unitsPerEm)
	}
	if ttf2.maxp.numGlyphs != ttf.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: %d vs %d", ttf2.maxp.numGlyphs, ttf.maxp.numGlyphs)
	}
	if ttf2.head.xMin != ttf.head.xMin || ttf2.head.yMin != ttf.head.yMin {
		t.Errorf("bbox min mismatch: (%d,%d) vs (%d,%d)", ttf2.head.xMin, ttf2.head.yMin, ttf.head.xMin, ttf.head.yMin)
	}
	if ttf2.head.xMax != ttf.head.xMax || ttf2.head.yMax != ttf.head.yMax {
		t.Errorf("bbox max mismatch: (%d,%d) vs (%d,%d)", ttf2.head.xMax, ttf2.head.yMax, ttf.head.xMax, ttf.head.yMax)
	}
}

func TestRoundTripEOT(t *testing.T) {
	// TTF → SerializeEOT → ParseEOT → SerializeEOT → ParseEOT → compare
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	eotData, err := ttf.SerializeEOT()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := ParseEOT(eotData)
	if err != nil {
		t.Fatal(err)
	}

	eotData2, err := ttf2.SerializeEOT()
	if err != nil {
		t.Fatal(err)
	}

	ttf3, err := ParseEOT(eotData2)
	if err != nil {
		t.Fatal(err)
	}

	if ttf3.head.unitsPerEm != ttf.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: %d vs %d", ttf3.head.unitsPerEm, ttf.head.unitsPerEm)
	}
	if ttf3.maxp.numGlyphs != ttf.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: %d vs %d", ttf3.maxp.numGlyphs, ttf.maxp.numGlyphs)
	}
}

func TestEOTToTTF(t *testing.T) {
	// TTF → SerializeEOT → ParseEOT → Serialize → Parse → compare with original
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	eotData, err := ttf.SerializeEOT()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := ParseEOT(eotData)
	if err != nil {
		t.Fatal(err)
	}

	ttfBytes, err := ttf2.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf3, err := Parse(ttfBytes)
	if err != nil {
		t.Fatal(err)
	}

	if ttf3.head.unitsPerEm != ttf.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: %d vs %d", ttf3.head.unitsPerEm, ttf.head.unitsPerEm)
	}
	if ttf3.maxp.numGlyphs != ttf.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: %d vs %d", ttf3.maxp.numGlyphs, ttf.maxp.numGlyphs)
	}
	if ttf3.head.xMin != ttf.head.xMin || ttf3.head.yMin != ttf.head.yMin {
		t.Errorf("bbox min mismatch")
	}
	if ttf3.head.xMax != ttf.head.xMax || ttf3.head.yMax != ttf.head.yMax {
		t.Errorf("bbox max mismatch")
	}
}
