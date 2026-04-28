package gofont

import (
	"testing"
)

func getHmtx(t *testing.T) *Hmtx {
	t.Helper()
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.hmtx == nil {
		t.Fatal("hmtx table is nil")
	}
	return ttf.hmtx
}

func TestParseHmtx(t *testing.T) {
	hmtx := getHmtx(t)

	if len(hmtx.hMetrics) != 43 {
		t.Fatalf("hMetrics count: got %d, want 43", len(hmtx.hMetrics))
	}
	if len(hmtx.leftSideBearing) != 0 {
		t.Errorf("leftSideBearing count: got %d, want 0 (numHMetrics == numGlyphs)", len(hmtx.leftSideBearing))
	}

	// First metric
	if hmtx.hMetrics[0].advanceWidth != 429 {
		t.Errorf("hMetrics[0].advanceWidth: got %d, want 429", hmtx.hMetrics[0].advanceWidth)
	}
	if hmtx.hMetrics[0].lsb != 50 {
		t.Errorf("hMetrics[0].lsb: got %d, want 50", hmtx.hMetrics[0].lsb)
	}

	// Last metric
	if hmtx.hMetrics[42].advanceWidth != 1354 {
		t.Errorf("hMetrics[42].advanceWidth: got %d, want 1354", hmtx.hMetrics[42].advanceWidth)
	}
	if hmtx.hMetrics[42].lsb != 8 {
		t.Errorf("hMetrics[42].lsb: got %d, want 8", hmtx.hMetrics[42].lsb)
	}
}

func TestRoundTripHmtx(t *testing.T) {
	hmtx := getHmtx(t)

	written := writeHmtx(hmtx)
	hmtx2, err := parseHmtx(written, len(hmtx.hMetrics), len(hmtx.hMetrics)+len(hmtx.leftSideBearing))
	if err != nil {
		t.Fatal(err)
	}

	if len(hmtx2.hMetrics) != len(hmtx.hMetrics) {
		t.Fatalf("hMetrics count mismatch: %d vs %d", len(hmtx2.hMetrics), len(hmtx.hMetrics))
	}
	for i, m := range hmtx.hMetrics {
		if hmtx2.hMetrics[i] != m {
			t.Errorf("hMetrics[%d] mismatch: got %+v, want %+v", i, hmtx2.hMetrics[i], m)
		}
	}
	if len(hmtx2.leftSideBearing) != len(hmtx.leftSideBearing) {
		t.Errorf("leftSideBearing count mismatch")
	}
	for i, lsb := range hmtx.leftSideBearing {
		if hmtx2.leftSideBearing[i] != lsb {
			t.Errorf("leftSideBearing[%d] mismatch", i)
		}
	}
}
