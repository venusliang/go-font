package gofont

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	woff2Data     []byte
	woff2DataOnce sync.Once
)

func loadWOFF2(t *testing.T) []byte {
	t.Helper()
	woff2DataOnce.Do(func() {
		paths := []string{
			"fonts/fonteditor.woff2",
			filepath.Join("..", "fonts", "fonteditor.woff2"),
		}
		var err error
		for _, p := range paths {
			woff2Data, err = os.ReadFile(p)
			if err == nil {
				return
			}
		}
		if woff2Data == nil {
			panic("fonts/fonteditor.woff2 not found: " + err.Error())
		}
	})
	return woff2Data
}

func TestParseWOFF2(t *testing.T) {
	ttf, err := ParseWOFF2(loadWOFF2(t))
	if err != nil {
		t.Fatal(err)
	}

	if ttf.head == nil {
		t.Fatal("head table is nil after WOFF2 parse")
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
	if ttf.glyf == nil {
		t.Fatal("glyf table is nil")
	}
	if ttf.loca == nil {
		t.Fatal("loca table is nil")
	}
}

func TestRoundTripWOFF2(t *testing.T) {
	// Parse WOFF2 → serialize to WOFF2 (no transform) → parse again
	ttf, err := ParseWOFF2(loadWOFF2(t))
	if err != nil {
		t.Fatal(err)
	}

	woff2Bytes, err := ttf.SerializeWOFF2()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := ParseWOFF2(woff2Bytes)
	if err != nil {
		t.Fatal(err)
	}

	if ttf2.head.unitsPerEm != ttf.head.unitsPerEm {
		t.Errorf("unitsPerEm mismatch: %d vs %d", ttf2.head.unitsPerEm, ttf.head.unitsPerEm)
	}
	if ttf2.maxp.numGlyphs != ttf.maxp.numGlyphs {
		t.Errorf("numGlyphs mismatch: %d vs %d", ttf2.maxp.numGlyphs, ttf.maxp.numGlyphs)
	}
}

func TestTTFToWOFF2(t *testing.T) {
	// Parse TTF → serialize to WOFF2 → parse WOFF2 → compare
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	woff2Bytes, err := ttf.SerializeWOFF2()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := ParseWOFF2(woff2Bytes)
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

func TestWOFF2ToTTF(t *testing.T) {
	// Parse WOFF2 → serialize to TTF → parse TTF → compare
	ttf, err := ParseWOFF2(loadWOFF2(t))
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

	// Compare with original TTF
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
	if ttf2.head.xMin != origTTF.head.xMin || ttf2.head.yMin != origTTF.head.yMin {
		t.Errorf("bbox min mismatch")
	}
	if ttf2.head.xMax != origTTF.head.xMax || ttf2.head.yMax != origTTF.head.yMax {
		t.Errorf("bbox max mismatch")
	}
}
