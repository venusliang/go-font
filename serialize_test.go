package gofonts

import (
	"testing"
)

func TestSerialize(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	// Parse the serialized font
	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse serialized font: %v", err)
	}

	// Verify all tables parsed correctly
	if ttf2.head == nil {
		t.Error("head is nil after round-trip")
	} else {
		if ttf2.head.magicNumber != 0x5F0F3CF5 {
			t.Errorf("head.magicNumber: got 0x%08X, want 0x5F0F3CF5", ttf2.head.magicNumber)
		}
		if ttf2.head.unitsPerEm != ttf.head.unitsPerEm {
			t.Errorf("head.unitsPerEm mismatch")
		}
	}

	if ttf2.maxp == nil {
		t.Error("maxp is nil after round-trip")
	} else {
		if ttf2.maxp.numGlyphs != ttf.maxp.numGlyphs {
			t.Errorf("maxp.numGlyphs: got %d, want %d", ttf2.maxp.numGlyphs, ttf.maxp.numGlyphs)
		}
	}

	if ttf2.os2 == nil {
		t.Error("os2 is nil after round-trip")
	} else {
		if ttf2.os2.version != ttf.os2.version {
			t.Errorf("os2.version mismatch")
		}
		if ttf2.os2.usWeightClass != ttf.os2.usWeightClass {
			t.Errorf("os2.usWeightClass mismatch")
		}
	}

	if ttf2.hhea == nil {
		t.Error("hhea is nil after round-trip")
	} else {
		if ttf2.hhea.numberOfHMetrics != ttf.hhea.numberOfHMetrics {
			t.Errorf("hhea.numberOfHMetrics mismatch")
		}
	}

	if ttf2.hmtx == nil {
		t.Error("hmtx is nil after round-trip")
	} else {
		if len(ttf2.hmtx.hMetrics) != len(ttf.hmtx.hMetrics) {
			t.Errorf("hmtx.hMetrics count mismatch")
		}
	}

	if ttf2.loca == nil {
		t.Error("loca is nil after round-trip")
	} else {
		if len(ttf2.loca.offsets) != len(ttf.loca.offsets) {
			t.Errorf("loca.offsets count mismatch")
		}
	}

	if ttf2.glyf == nil {
		t.Error("glyf is nil after round-trip")
	} else {
		if len(ttf2.glyf) != len(ttf.glyf) {
			t.Errorf("glyf count: got %d, want %d", len(ttf2.glyf), len(ttf.glyf))
		}
		// Verify first glyph coordinates match
		if len(ttf2.glyf) > 0 && ttf2.glyf[0].simpleGlyph != nil && ttf.glyf[0].simpleGlyph != nil {
			sg1 := ttf.glyf[0].simpleGlyph
			sg2 := ttf2.glyf[0].simpleGlyph
			if len(sg2.xCoordinates) != len(sg1.xCoordinates) {
				t.Errorf("glyph[0] point count mismatch")
			} else {
				for j, x := range sg1.xCoordinates {
					if sg2.xCoordinates[j] != x {
						t.Errorf("glyph[0] x[%d] mismatch: got %d, want %d", j, sg2.xCoordinates[j], x)
						break
					}
				}
			}
		}
	}

	if ttf2.cmap == nil {
		t.Error("cmap is nil after round-trip")
	}

	if ttf2.post == nil {
		t.Error("post is nil after round-trip")
	}

	if ttf2.name == nil {
		t.Error("name is nil after round-trip")
	}
}

func TestSerializeMap(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatal(err)
	}

	// Verify cmap mapping works on serialized font
	for _, sub := range ttf2.cmap.subtables {
		if sub.Format() == 4 {
			f4 := sub.(*CMapFormat4)
			if gid := f4.Map(0xE001); gid != 1 {
				t.Errorf("Map(0xE001): got %d, want 1", gid)
			}
			if gid := f4.Map(0xE030); gid != 42 {
				t.Errorf("Map(0xE030): got %d, want 42", gid)
			}
		}
	}
}
