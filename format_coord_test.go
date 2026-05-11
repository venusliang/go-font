package gofont

import (
	"testing"
)

// compareGlyphCoordinates compares all glyph coordinates between two fonts.
// Returns the number of coordinate mismatches.
func compareGlyphCoordinates(t *testing.T, orig, result TrueTypeFont) int {
	t.Helper()
	numOrig := len(orig.glyf)
	numResult := len(result.glyf)
	if numOrig != numResult {
		t.Errorf("glyph count mismatch: orig=%d, result=%d", numOrig, numResult)
		return -1
	}

	mismatches := 0
	for i := 0; i < numOrig; i++ {
		og := orig.glyf[i]
		rg := result.glyf[i]
		if og == nil && rg == nil {
			continue
		}
		if og == nil || rg == nil {
			t.Errorf("glyph %d: one is nil", i)
			mismatches++
			continue
		}

		if og.header != rg.header {
			if mismatches < 3 {
				t.Errorf("glyph %d header: orig=%+v result=%+v", i, og.header, rg.header)
			}
			mismatches++
		}

		if og.simpleGlyph != nil && rg.simpleGlyph != nil {
			osg := og.simpleGlyph
			rsg := rg.simpleGlyph
			if len(osg.xCoordinates) != len(rsg.xCoordinates) {
				t.Errorf("glyph %d point count: orig=%d result=%d", i, len(osg.xCoordinates), len(rsg.xCoordinates))
				mismatches++
				continue
			}
			for j := 0; j < len(osg.xCoordinates); j++ {
				if osg.xCoordinates[j] != rsg.xCoordinates[j] || osg.yCoordinates[j] != rsg.yCoordinates[j] {
					if mismatches < 5 {
						t.Errorf("glyph %d pt %d: orig=(%d,%d) result=(%d,%d)", i, j,
							osg.xCoordinates[j], osg.yCoordinates[j],
							rsg.xCoordinates[j], rsg.yCoordinates[j])
					}
					mismatches++
				}
			}
		}

		if og.compositeGlyph != nil && rg.compositeGlyph != nil {
			ocg := og.compositeGlyph
			rcg := rg.compositeGlyph
			if len(ocg.components) != len(rcg.components) {
				t.Errorf("glyph %d component count: orig=%d result=%d", i, len(ocg.components), len(rcg.components))
				mismatches++
				continue
			}
			for j := 0; j < len(ocg.components); j++ {
				oc := ocg.components[j]
				rc := rcg.components[j]
				if oc.glyphIndex != rc.glyphIndex || oc.arg1 != rc.arg1 || oc.arg2 != rc.arg2 {
					if mismatches < 5 {
						t.Errorf("glyph %d comp %d: orig(idx=%d,args=(%d,%d)) result(idx=%d,args=(%d,%d))",
							i, j, oc.glyphIndex, oc.arg1, oc.arg2, rc.glyphIndex, rc.arg1, rc.arg2)
					}
					mismatches++
				}
			}
		}
	}
	return mismatches
}

func TestWOFFToTTFCoordinates(t *testing.T) {
	woffParsed, err := ParseWOFF(loadWOFF(t))
	if err != nil {
		t.Fatal(err)
	}
	ttfBytes, err := woffParsed.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	resultTTF, err := Parse(ttfBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Compare: WOFF font → TTF serialize/parse should preserve glyphs
	mismatches := compareGlyphCoordinates(t, woffParsed, resultTTF)
	if mismatches != 0 {
		t.Errorf("WOFF→TTF: %d coordinate mismatches", mismatches)
	}
}

func TestTTFSerdeToWOFFCoordinates(t *testing.T) {
	origTTF, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	woffBytes, err := origTTF.SerializeWOFF()
	if err != nil {
		t.Fatal(err)
	}
	woffParsed, err := ParseWOFF(woffBytes)
	if err != nil {
		t.Fatal(err)
	}
	ttfBytes, err := woffParsed.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	resultTTF, err := Parse(ttfBytes)
	if err != nil {
		t.Fatal(err)
	}

	mismatches := compareGlyphCoordinates(t, origTTF, resultTTF)
	if mismatches != 0 {
		t.Errorf("TTF→WOFF→TTF: %d coordinate mismatches", mismatches)
	}
}

func TestEOTToTTFCoordinates(t *testing.T) {
	origTTF, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	eotData, err := origTTF.SerializeEOT()
	if err != nil {
		t.Fatal(err)
	}
	eotParsed, err := ParseEOT(eotData)
	if err != nil {
		t.Fatal(err)
	}
	ttfBytes, err := eotParsed.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	resultTTF, err := Parse(ttfBytes)
	if err != nil {
		t.Fatal(err)
	}

	mismatches := compareGlyphCoordinates(t, origTTF, resultTTF)
	if mismatches != 0 {
		t.Errorf("TTF→EOT→TTF: %d coordinate mismatches", mismatches)
	}
}

func TestWOFF2ToTTFCoordinates(t *testing.T) {
	woff2Parsed, err := ParseWOFF2(loadWOFF2(t))
	if err != nil {
		t.Fatal(err)
	}
	ttfBytes, err := woff2Parsed.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	resultTTF, err := Parse(ttfBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Compare: WOFF2 font → TTF serialize/parse should preserve glyphs
	mismatches := compareGlyphCoordinates(t, woff2Parsed, resultTTF)
	if mismatches != 0 {
		t.Errorf("WOFF2→TTF: %d coordinate mismatches", mismatches)
	}
}

func TestTTCSerdeCoordinates(t *testing.T) {
	origTTF, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	ttcData, err := SerializeTTC([]TrueTypeFont{origTTF})
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
	resultTTF, err := Parse(ttfBytes)
	if err != nil {
		t.Fatal(err)
	}

	mismatches := compareGlyphCoordinates(t, origTTF, resultTTF)
	if mismatches != 0 {
		t.Errorf("TTF→TTC→TTF: %d coordinate mismatches", mismatches)
	}
}

func TestCrossFormatCoordinates(t *testing.T) {
	// Full chain: TTF → WOFF → WOFF2 → EOT → TTF
	origTTF, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// TTF → WOFF → TTF
	woffBytes, err := origTTF.SerializeWOFF()
	if err != nil {
		t.Fatal(err)
	}
	woffFont, err := ParseWOFF(woffBytes)
	if err != nil {
		t.Fatal(err)
	}
	m := compareGlyphCoordinates(t, origTTF, woffFont)
	if m != 0 {
		t.Errorf("After WOFF: %d mismatches", m)
	}

	// WOFF → WOFF2 → TTF (WOFF2 uses no glyf transform since source is raw tables)
	woff2Bytes, err := woffFont.SerializeWOFF2()
	if err != nil {
		t.Fatal(err)
	}
	woff2Font, err := ParseWOFF2(woff2Bytes)
	if err != nil {
		t.Fatal(err)
	}
	m = compareGlyphCoordinates(t, origTTF, woff2Font)
	if m != 0 {
		t.Errorf("After WOFF2: %d mismatches", m)
	}

	// WOFF2 → EOT → TTF
	eotBytes, err := woff2Font.SerializeEOT()
	if err != nil {
		t.Fatal(err)
	}
	eotFont, err := ParseEOT(eotBytes)
	if err != nil {
		t.Fatal(err)
	}
	m = compareGlyphCoordinates(t, origTTF, eotFont)
	if m != 0 {
		t.Errorf("After EOT: %d mismatches", m)
	}

	// Final: EOT → TTF
	ttfBytes, err := eotFont.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	finalTTF, err := Parse(ttfBytes)
	if err != nil {
		t.Fatal(err)
	}
	m = compareGlyphCoordinates(t, origTTF, finalTTF)
	if m != 0 {
		t.Errorf("After final TTF: %d mismatches", m)
	}
}
