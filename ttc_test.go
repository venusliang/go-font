package gofont

import (
	"testing"
)

func TestTTFToTTC(t *testing.T) {
	// TTF → Parse → SerializeTTC → ParseTTC → compare each font
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Create a TTC with 2 copies of the same font
	ttcData, err := SerializeTTC([]TrueTypeFont{ttf, ttf})
	if err != nil {
		t.Fatal(err)
	}

	fonts, err := ParseTTC(ttcData)
	if err != nil {
		t.Fatal(err)
	}

	if len(fonts) != 2 {
		t.Fatalf("expected 2 fonts, got %d", len(fonts))
	}

	for i, f := range fonts {
		if f.head.unitsPerEm != ttf.head.unitsPerEm {
			t.Errorf("font %d: unitsPerEm mismatch: %d vs %d", i, f.head.unitsPerEm, ttf.head.unitsPerEm)
		}
		if f.maxp.numGlyphs != ttf.maxp.numGlyphs {
			t.Errorf("font %d: numGlyphs mismatch: %d vs %d", i, f.maxp.numGlyphs, ttf.maxp.numGlyphs)
		}
		if f.head.xMin != ttf.head.xMin || f.head.yMin != ttf.head.yMin {
			t.Errorf("font %d: bbox min mismatch: (%d,%d) vs (%d,%d)", i, f.head.xMin, f.head.yMin, ttf.head.xMin, ttf.head.yMin)
		}
		if f.head.xMax != ttf.head.xMax || f.head.yMax != ttf.head.yMax {
			t.Errorf("font %d: bbox max mismatch: (%d,%d) vs (%d,%d)", i, f.head.xMax, f.head.yMax, ttf.head.xMax, ttf.head.yMax)
		}
	}
}

func TestRoundTripTTC(t *testing.T) {
	// TTF → SerializeTTC → ParseTTC → SerializeTTC → ParseTTC → compare
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	ttcData, err := SerializeTTC([]TrueTypeFont{ttf})
	if err != nil {
		t.Fatal(err)
	}

	fonts, err := ParseTTC(ttcData)
	if err != nil {
		t.Fatal(err)
	}

	ttcData2, err := SerializeTTC(fonts)
	if err != nil {
		t.Fatal(err)
	}

	fonts2, err := ParseTTC(ttcData2)
	if err != nil {
		t.Fatal(err)
	}

	if len(fonts2) != 1 {
		t.Fatalf("expected 1 font, got %d", len(fonts2))
	}

	f := fonts2[0]
	if f.head.unitsPerEm != ttf.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: %d vs %d", f.head.unitsPerEm, ttf.head.unitsPerEm)
	}
	if f.maxp.numGlyphs != ttf.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: %d vs %d", f.maxp.numGlyphs, ttf.maxp.numGlyphs)
	}
}

// TestFormatCrossRoundTrip chains all formats in sequence:
// TTF → TTC → TTF → WOFF → WOFF2 → EOT → TTF
// Each step extracts, converts to the next format, then re-parses.
// The original TTF values are checked at the end.
func TestFormatCrossRoundTrip(t *testing.T) {
	// Step 0: Parse original TTF
	original, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal("parse original:", err)
	}

	assertFontEquals := func(t *testing.T, got TrueTypeFont, step string) {
		t.Helper()
		if got.head.unitsPerEm != original.head.unitsPerEm {
			t.Errorf("%s: unitsPerEm mismatch: %d vs %d", step, got.head.unitsPerEm, original.head.unitsPerEm)
		}
		if got.maxp.numGlyphs != original.maxp.numGlyphs {
			t.Errorf("%s: numGlyphs mismatch: %d vs %d", step, got.maxp.numGlyphs, original.maxp.numGlyphs)
		}
		if got.head.xMin != original.head.xMin || got.head.yMin != original.head.yMin {
			t.Errorf("%s: bbox min mismatch: (%d,%d) vs (%d,%d)", step, got.head.xMin, got.head.yMin, original.head.xMin, original.head.yMin)
		}
		if got.head.xMax != original.head.xMax || got.head.yMax != original.head.yMax {
			t.Errorf("%s: bbox max mismatch: (%d,%d) vs (%d,%d)", step, got.head.xMax, got.head.yMax, original.head.xMax, original.head.yMax)
		}
	}

	// Step 1: TTF → SerializeTTC → ParseTTC (extract first font)
	ttcData, err := SerializeTTC([]TrueTypeFont{original})
	if err != nil {
		t.Fatal("serialize TTC:", err)
	}
	ttcFonts, err := ParseTTC(ttcData)
	if err != nil {
		t.Fatal("parse TTC:", err)
	}
	if len(ttcFonts) != 1 {
		t.Fatalf("TTC: expected 1 font, got %d", len(ttcFonts))
	}
	assertFontEquals(t, ttcFonts[0], "TTC→extract")

	// Step 2: TTC font → Serialize → Parse (back to TTF)
	ttfData, err := ttcFonts[0].Serialize()
	if err != nil {
		t.Fatal("serialize TTF:", err)
	}
	ttfFont, err := Parse(ttfData)
	if err != nil {
		t.Fatal("parse TTF:", err)
	}
	assertFontEquals(t, ttfFont, "TTF")

	// Step 3: TTF → SerializeWOFF → ParseWOFF
	woffData, err := ttfFont.SerializeWOFF()
	if err != nil {
		t.Fatal("serialize WOFF:", err)
	}
	woffFont, err := ParseWOFF(woffData)
	if err != nil {
		t.Fatal("parse WOFF:", err)
	}
	assertFontEquals(t, woffFont, "WOFF")

	// Step 4: WOFF → SerializeWOFF2 → ParseWOFF2
	woff2Data, err := woffFont.SerializeWOFF2()
	if err != nil {
		t.Fatal("serialize WOFF2:", err)
	}
	woff2Font, err := ParseWOFF2(woff2Data)
	if err != nil {
		t.Fatal("parse WOFF2:", err)
	}
	assertFontEquals(t, woff2Font, "WOFF2")

	// Step 5: WOFF2 → SerializeEOT → ParseEOT
	eotData, err := woff2Font.SerializeEOT()
	if err != nil {
		t.Fatal("serialize EOT:", err)
	}
	eotFont, err := ParseEOT(eotData)
	if err != nil {
		t.Fatal("parse EOT:", err)
	}
	assertFontEquals(t, eotFont, "EOT")

	// Step 6: EOT → Serialize → Parse (back to TTF, full circle)
	finalTTF, err := eotFont.Serialize()
	if err != nil {
		t.Fatal("serialize final TTF:", err)
	}
	finalFont, err := Parse(finalTTF)
	if err != nil {
		t.Fatal("parse final TTF:", err)
	}
	assertFontEquals(t, finalFont, "TTF(final)")
}

func TestTTCToTTF(t *testing.T) {
	// TTF → SerializeTTC → ParseTTC → Serialize (single font) → Parse → compare with original
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	ttcData, err := SerializeTTC([]TrueTypeFont{ttf})
	if err != nil {
		t.Fatal(err)
	}

	fonts, err := ParseTTC(ttcData)
	if err != nil {
		t.Fatal(err)
	}

	ttfBytes, err := fonts[0].Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(ttfBytes)
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
		t.Errorf("bbox min mismatch")
	}
	if ttf2.head.xMax != ttf.head.xMax || ttf2.head.yMax != ttf.head.yMax {
		t.Errorf("bbox max mismatch")
	}
}
