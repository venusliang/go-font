package gofonts

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

	// 43 glyphs + 1 end entry = 44 offsets
	if len(loca.offsets) != 44 {
		t.Fatalf("offsets count: got %d, want 44", len(loca.offsets))
	}

	// glyph[0] starts at offset 0
	if loca.offsets[0] != 0 {
		t.Errorf("offsets[0]: got %d, want 0", loca.offsets[0])
	}

	// glyph[1] at offset 40
	if loca.offsets[1] != 40 {
		t.Errorf("offsets[1]: got %d, want 40", loca.offsets[1])
	}

	// last entry = glyf table size
	if loca.offsets[43] != 6564 {
		t.Errorf("offsets[43]: got %d, want 6564", loca.offsets[43])
	}
}

func TestRoundTripLoca(t *testing.T) {
	loca := getLoca(t)

	written := writeLoca(loca, 0) // short format
	loca2, err := parseLoca(written, 43, 0)
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
