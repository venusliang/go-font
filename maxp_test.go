package gofonts

import (
	"testing"
)

func TestParseMaxp(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	maxp := ttf.maxp
	if maxp == nil {
		t.Fatal("maxp table is nil")
	}

	// NOTE: maxp version is read as U16 but should be U32 (Fixed 16.16).
	// After bug fix, version should be 0x00010000.
	// The current code reads 0x0001, which shifts all subsequent fields by 2 bytes.
	// These assertions reflect the CORRECT values after the bug is fixed.
	if maxp.numGlyphs != 43 {
		t.Errorf("numGlyphs: got %d, want 43", maxp.numGlyphs)
	}
	if maxp.maxPoints != 186 {
		t.Errorf("maxPoints: got %d, want 186", maxp.maxPoints)
	}
	if maxp.maxContours != 13 {
		t.Errorf("maxContours: got %d, want 13", maxp.maxContours)
	}
	if maxp.maxZones != 2 {
		t.Errorf("maxZones: got %d, want 2", maxp.maxZones)
	}
}

func TestRoundTripMaxp(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	written := writeMaxp(ttf.maxp)
	maxp2, err := parseMaxp(written)
	if err != nil {
		t.Fatal(err)
	}

	orig := ttf.maxp
	if maxp2.numGlyphs != orig.numGlyphs {
		t.Errorf("numGlyphs mismatch: %d vs %d", maxp2.numGlyphs, orig.numGlyphs)
	}
	if maxp2.maxPoints != orig.maxPoints {
		t.Errorf("maxPoints mismatch")
	}
	if maxp2.maxContours != orig.maxContours {
		t.Errorf("maxContours mismatch")
	}
	if maxp2.maxCompositePoints != orig.maxCompositePoints {
		t.Errorf("maxCompositePoints mismatch")
	}
	if maxp2.maxCompositeContours != orig.maxCompositeContours {
		t.Errorf("maxCompositeContours mismatch")
	}
	if maxp2.maxZones != orig.maxZones {
		t.Errorf("maxZones mismatch")
	}
	if maxp2.maxTwilightPoints != orig.maxTwilightPoints {
		t.Errorf("maxTwilightPoints mismatch")
	}
	if maxp2.maxStorage != orig.maxStorage {
		t.Errorf("maxStorage mismatch")
	}
	if maxp2.maxFunctionDefs != orig.maxFunctionDefs {
		t.Errorf("maxFunctionDefs mismatch")
	}
	if maxp2.maxStackElements != orig.maxStackElements {
		t.Errorf("maxStackElements mismatch")
	}
}
