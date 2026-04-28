package gofont

import (
	"testing"
)

func getHhea(t *testing.T) *Hhea {
	t.Helper()
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.hhea == nil {
		t.Fatal("hhea table is nil")
	}
	return ttf.hhea
}

func TestParseHhea(t *testing.T) {
	hhea := getHhea(t)

	if hhea.version != 0x00010000 {
		t.Errorf("version: got 0x%08X, want 0x00010000", hhea.version)
	}
	if hhea.ascent != 812 {
		t.Errorf("ascent: got %d, want 812", hhea.ascent)
	}
	if hhea.descent != -212 {
		t.Errorf("descent: got %d, want -212", hhea.descent)
	}
	if hhea.lineGap != 92 {
		t.Errorf("lineGap: got %d, want 92", hhea.lineGap)
	}
	if hhea.advanceWidthMax != 1354 {
		t.Errorf("advanceWidthMax: got %d, want 1354", hhea.advanceWidthMax)
	}
	if hhea.minLeftSideBearing != 6 {
		t.Errorf("minLeftSideBearing: got %d, want 6", hhea.minLeftSideBearing)
	}
	if hhea.minRightSideBearing != 29 {
		t.Errorf("minRightSideBearing: got %d, want 29", hhea.minRightSideBearing)
	}
	if hhea.xMaxExtent != 1321 {
		t.Errorf("xMaxExtent: got %d, want 1321", hhea.xMaxExtent)
	}
	if hhea.caretSlopeRise != 1 {
		t.Errorf("caretSlopeRise: got %d, want 1", hhea.caretSlopeRise)
	}
	if hhea.caretSlopeRun != 0 {
		t.Errorf("caretSlopeRun: got %d, want 0", hhea.caretSlopeRun)
	}
	if hhea.numberOfHMetrics != 43 {
		t.Errorf("numberOfHMetrics: got %d, want 43", hhea.numberOfHMetrics)
	}
	if hhea.metricDataFormat != 0 {
		t.Errorf("metricDataFormat: got %d, want 0", hhea.metricDataFormat)
	}
}

func TestRoundTripHhea(t *testing.T) {
	hhea := getHhea(t)

	written := writeHhea(hhea)
	hhea2, err := parseHhea(written)
	if err != nil {
		t.Fatal(err)
	}

	if hhea2.version != hhea.version {
		t.Errorf("version mismatch")
	}
	if hhea2.ascent != hhea.ascent {
		t.Errorf("ascent mismatch")
	}
	if hhea2.descent != hhea.descent {
		t.Errorf("descent mismatch")
	}
	if hhea2.lineGap != hhea.lineGap {
		t.Errorf("lineGap mismatch")
	}
	if hhea2.advanceWidthMax != hhea.advanceWidthMax {
		t.Errorf("advanceWidthMax mismatch")
	}
	if hhea2.minLeftSideBearing != hhea.minLeftSideBearing {
		t.Errorf("minLeftSideBearing mismatch")
	}
	if hhea2.minRightSideBearing != hhea.minRightSideBearing {
		t.Errorf("minRightSideBearing mismatch")
	}
	if hhea2.xMaxExtent != hhea.xMaxExtent {
		t.Errorf("xMaxExtent mismatch")
	}
	if hhea2.numberOfHMetrics != hhea.numberOfHMetrics {
		t.Errorf("numberOfHMetrics mismatch")
	}
}
