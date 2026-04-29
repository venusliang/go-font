package gofont

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	woffData     []byte
	woffDataOnce sync.Once
)

func loadWOFF(t *testing.T) []byte {
	t.Helper()
	woffDataOnce.Do(func() {
		paths := []string{
			"fonts/fonteditor.woff",
			filepath.Join("..", "fonts", "fonteditor.woff"),
		}
		var err error
		for _, p := range paths {
			woffData, err = os.ReadFile(p)
			if err == nil {
				return
			}
		}
		if woffData == nil {
			panic("fonts/fonteditor.woff not found: " + err.Error())
		}
	})
	return woffData
}

func TestParseWOFF(t *testing.T) {
	ttf, err := ParseWOFF(loadWOFF(t))
	if err != nil {
		t.Fatal(err)
	}

	// Verify basic TTF parsing worked
	if ttf.head == nil {
		t.Fatal("head table is nil after WOFF parse")
	}
	if ttf.head.unitsPerEm != 1024 {
		t.Errorf("unitsPerEm: got %d, want 1024", ttf.head.unitsPerEm)
	}
	if ttf.maxp == nil {
		t.Fatal("maxp table is nil")
	}
	if ttf.cmap == nil {
		t.Fatal("cmap table is nil")
	}
}

func TestRoundTripWOFF(t *testing.T) {
	// Parse WOFF → serialize to WOFF → parse again
	ttf, err := ParseWOFF(loadWOFF(t))
	if err != nil {
		t.Fatal(err)
	}

	woffBytes, err := ttf.SerializeWOFF()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := ParseWOFF(woffBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Verify key tables survived the round trip
	if ttf2.head.unitsPerEm != ttf.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: %d vs %d", ttf2.head.unitsPerEm, ttf.head.unitsPerEm)
	}
	if ttf2.maxp.numGlyphs != ttf.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: %d vs %d", ttf2.maxp.numGlyphs, ttf.maxp.numGlyphs)
	}
}

func TestTTFSerdeToWOFF(t *testing.T) {
	// Parse TTF → serialize to WOFF → parse WOFF → compare
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	woffBytes, err := ttf.SerializeWOFF()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := ParseWOFF(woffBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Verify tables match
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

func TestWOFFToTTF(t *testing.T) {
	// Parse WOFF → serialize to TTF → parse TTF → compare
	ttf, err := ParseWOFF(loadWOFF(t))
	if err != nil {
		t.Fatal(err)
	}

	ttfBytes, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(ttfBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the TTF produced from WOFF matches the original TTF
	origTTF, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	if ttf2.head.unitsPerEm != origTTF.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: %d vs %d", ttf2.head.unitsPerEm, origTTF.head.unitsPerEm)
	}
	if ttf2.maxp.numGlyphs != origTTF.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: %d vs %d", ttf2.maxp.numGlyphs, origTTF.maxp.numGlyphs)
	}
}
