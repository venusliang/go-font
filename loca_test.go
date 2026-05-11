package gofont

import (
	"testing"
)

func getLoca(t *testing.T) *Loca {
	t.Helper()
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.loca == nil {
		t.Fatal("loca table is nil")
	}
	return ttf.loca
}

func TestParseLoca(t *testing.T) {
	loca := getLoca(t)

	numGlyphs := len(loca.offsets)
	if numGlyphs < 2 {
		t.Fatalf("offsets count: got %d, want at least 2", numGlyphs)
	}

	// glyph[0] starts at offset 0
	if loca.offsets[0] != 0 {
		t.Errorf("offsets[0]: got %d, want 0", loca.offsets[0])
	}

	// offsets should be non-decreasing
	for i := 1; i < len(loca.offsets); i++ {
		if loca.offsets[i] < loca.offsets[i-1] {
			t.Errorf("offsets[%d]=%d < offsets[%d]=%d", i, loca.offsets[i], i-1, loca.offsets[i-1])
		}
	}
}

func TestRoundTripLoca(t *testing.T) {
	loca := getLoca(t)
	numGlyphs := len(loca.offsets) - 1
	// Use the font's indexToLocFormat to determine the format
	indexToLocFormat := int16(1) // Microsoft YaHei uses long format

	written := writeLoca(loca, indexToLocFormat)
	loca2, err := parseLoca(written, numGlyphs, indexToLocFormat)
	if err != nil {
		t.Fatal(err)
	}

	if len(loca2.offsets) != len(loca.offsets) {
		t.Fatalf("offsets count mismatch: %d vs %d", len(loca2.offsets), len(loca.offsets))
	}
	for i, off := range loca.offsets {
		if loca2.offsets[i] != off {
			t.Errorf("offsets[%d] mismatch: got %d, want %d", i, loca2.offsets[i], off)
		}
	}
}
