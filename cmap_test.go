package gofont

import (
	"testing"
)

func getCmap(t *testing.T) *CMap {
	t.Helper()
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.cmap == nil {
		t.Fatal("cmap table is nil")
	}
	return ttf.cmap
}

func TestParseCmap(t *testing.T) {
	cmap := getCmap(t)

	if cmap.version != 0 {
		t.Errorf("version: got %d, want 0", cmap.version)
	}
	if cmap.numTables != 2 {
		t.Errorf("numTables: got %d, want 2", cmap.numTables)
	}
	if len(cmap.encodingRecords) != 2 {
		t.Errorf("encodingRecords count: got %d, want 2", len(cmap.encodingRecords))
	}

	// Check encoding records
	if cmap.encodingRecords[0].platformID != 0 {
		t.Errorf("record[0] platformID: got %d, want 0", cmap.encodingRecords[0].platformID)
	}
	if cmap.encodingRecords[1].platformID != 3 {
		t.Errorf("record[1] platformID: got %d, want 3", cmap.encodingRecords[1].platformID)
	}

	// Check subtables were parsed
	if len(cmap.subtables) < 2 {
		t.Fatalf("subtables count: got %d, want at least 2", len(cmap.subtables))
	}
}

func TestCmapFormat4Map(t *testing.T) {
	cmap := getCmap(t)

	// Find a format 4 subtable
	var f4 *CMapFormat4
	for _, sub := range cmap.subtables {
		if sub.Format() == 4 {
			f4 = sub.(*CMapFormat4)
			break
		}
	}
	if f4 == nil {
		t.Fatal("no format 4 subtable found")
	}

	// U+0020 (space) should map to glyph 3
	if gid := f4.Map(0x0020); gid != 3 {
		t.Errorf("Map(0x0020): got %d, want 3", gid)
	}

	// U+0041 ('A') should map to glyph 39
	if gid := f4.Map(0x0041); gid != 39 {
		t.Errorf("Map(0x0041): got %d, want 39", gid)
	}

	// U+4E00 (CJK '一') should map to glyph 9650
	if gid := f4.Map(0x4E00); gid != 9650 {
		t.Errorf("Map(0x4E00): got %d, want 9650", gid)
	}

	// U+E001 should not be mapped in this font
	if gid := f4.Map(0xE001); gid != 0 {
		t.Errorf("Map(0xE001): got %d, want 0 (not mapped)", gid)
	}
}

func TestCmapFormat0Map(t *testing.T) {
	cmap := getCmap(t)

	// Find format 0 subtable (if present)
	var f0 *CMapFormat0
	for _, sub := range cmap.subtables {
		if sub.Format() == 0 {
			f0 = sub.(*CMapFormat0)
			break
		}
	}
	if f0 == nil {
		t.Log("no format 0 subtable in this font (not all fonts have one)")
		return
	}

	// Check that format 0 has 256 byte entries
	if f0.glyphIdArray[0] != 0 {
		t.Errorf("glyphIdArray[0]: got %d, want 0", f0.glyphIdArray[0])
	}
}

func TestRoundTripCmap(t *testing.T) {
	cmap := getCmap(t)

	written := writeCmap(cmap)
	cmap2, err := parseCmap(TableDirectory{}, written)
	if err != nil {
		t.Fatal(err)
	}

	if cmap2.version != cmap.version {
		t.Errorf("version mismatch")
	}
	if cmap2.numTables != cmap.numTables {
		t.Errorf("numTables mismatch")
	}
	if len(cmap2.encodingRecords) != len(cmap.encodingRecords) {
		t.Fatalf("encodingRecords count mismatch: %d vs %d", len(cmap2.encodingRecords), len(cmap.encodingRecords))
	}

	for i, rec := range cmap.encodingRecords {
		rec2 := cmap2.encodingRecords[i]
		if rec2.platformID != rec.platformID {
			t.Errorf("record[%d] platformID mismatch", i)
		}
		if rec2.encodingID != rec.encodingID {
			t.Errorf("record[%d] encodingID mismatch", i)
		}
	}

	// Verify mapping works on round-tripped cmap
	for _, sub := range cmap2.subtables {
		if sub.Format() == 4 {
			f4 := sub.(*CMapFormat4)
			if gid := f4.Map(0x0020); gid != 3 {
				t.Errorf("round-trip Map(0x0020): got %d, want 3", gid)
			}
			if gid := f4.Map(0x0041); gid != 39 {
				t.Errorf("round-trip Map(0x0041): got %d, want 39", gid)
			}
		}
	}
}
