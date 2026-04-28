package gofont

import (
	"testing"
)

func TestRuneToGlyphID(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Known mapping from test font
	if gid := ttf.RuneToGlyphID(0xE001); gid != 1 {
		t.Errorf("RuneToGlyphID(0xE001): got %d, want 1", gid)
	}
	if gid := ttf.RuneToGlyphID(0xE002); gid != 2 {
		t.Errorf("RuneToGlyphID(0xE002): got %d, want 2", gid)
	}
	if gid := ttf.RuneToGlyphID(0xE030); gid != 42 {
		t.Errorf("RuneToGlyphID(0xE030): got %d, want 42", gid)
	}

	// Unmapped rune should return 0
	if gid := ttf.RuneToGlyphID(0x41); gid != 0 {
		t.Errorf("RuneToGlyphID('A'): got %d, want 0", gid)
	}
}

func TestGlyphForRune(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	g := ttf.GlyphForRune(0xE001)
	if g == nil {
		t.Fatal("GlyphForRune(0xE001) returned nil")
	}
	if g.simpleGlyph == nil {
		t.Error("glyph 1 is not a simple glyph")
	}

	// Unmapped rune should return nil
	if g := ttf.GlyphForRune(0x41); g != nil {
		t.Error("GlyphForRune('A') should be nil")
	}
}

func TestSetRuneMapping(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Set a new mapping
	err = ttf.SetRuneMapping(0x41, 1)
	if err != nil {
		t.Fatal(err)
	}
	if gid := ttf.RuneToGlyphID(0x41); gid != 1 {
		t.Errorf("after SetRuneMapping(0x41, 1): got %d, want 1", gid)
	}

	// Override an existing mapping
	err = ttf.SetRuneMapping(0xE001, 5)
	if err != nil {
		t.Fatal(err)
	}
	if gid := ttf.RuneToGlyphID(0xE001); gid != 5 {
		t.Errorf("after SetRuneMapping(0xE001, 5): got %d, want 5", gid)
	}

	// Out of range glyph ID
	err = ttf.SetRuneMapping(0x42, 100)
	if err == nil {
		t.Error("expected error for out-of-range glyph ID")
	}
}

func TestRemoveRuneMapping(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Verify mapping exists
	if gid := ttf.RuneToGlyphID(0xE001); gid != 1 {
		t.Fatalf("before remove: got %d, want 1", gid)
	}

	ttf.RemoveRuneMapping(0xE001)
	if gid := ttf.RuneToGlyphID(0xE001); gid != 0 {
		t.Errorf("after RemoveRuneMapping(0xE001): got %d, want 0", gid)
	}

	// Removing non-existent mapping should be a no-op
	ttf.RemoveRuneMapping(0x42)
}

func TestNumGlyphs(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if n := ttf.NumGlyphs(); n != 43 {
		t.Errorf("NumGlyphs: got %d, want 43", n)
	}
}

func TestGlyphAt(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	g := ttf.GlyphAt(0)
	if g == nil {
		t.Fatal("GlyphAt(0) returned nil")
	}
	if g.header.numberOfContours != 2 {
		t.Errorf("glyph 0 numberOfContours: got %d, want 2", g.header.numberOfContours)
	}

	// Out of range
	if g := ttf.GlyphAt(-1); g != nil {
		t.Error("GlyphAt(-1) should be nil")
	}
	if g := ttf.GlyphAt(100); g != nil {
		t.Error("GlyphAt(100) should be nil")
	}
}

func TestSetGlyphAt(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Replace glyph 1 with a copy of glyph 0
	orig := ttf.GlyphAt(0)
	newGlyph := &Glyph{
		header: orig.header,
	}
	if orig.simpleGlyph != nil {
		sg := *orig.simpleGlyph
		newGlyph.simpleGlyph = &sg
	}

	err = ttf.SetGlyphAt(1, newGlyph)
	if err != nil {
		t.Fatal(err)
	}

	g := ttf.GlyphAt(1)
	if g.header.numberOfContours != orig.header.numberOfContours {
		t.Errorf("after SetGlyphAt: numberOfContours got %d, want %d", g.header.numberOfContours, orig.header.numberOfContours)
	}

	// Out of range
	err = ttf.SetGlyphAt(100, newGlyph)
	if err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestRemoveGlyphs(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	origCount := ttf.NumGlyphs()

	// Remove glyph 2 and glyph 5
	remap, err := ttf.RemoveGlyphs([]int{2, 5})
	if err != nil {
		t.Fatal(err)
	}

	// Check new glyph count
	if n := ttf.NumGlyphs(); n != origCount-2 {
		t.Errorf("NumGlyphs after remove: got %d, want %d", n, origCount-2)
	}

	// Check remap: glyph 0 stays at 0
	if remap[0] != 0 {
		t.Errorf("remap[0]: got %d, want 0", remap[0])
	}
	// glyph 1 stays at 1
	if remap[1] != 1 {
		t.Errorf("remap[1]: got %d, want 1", remap[1])
	}
	// glyph 2 was removed
	if _, ok := remap[2]; ok {
		t.Error("remap should not contain removed index 2")
	}
	// glyph 3 moves to 2
	if remap[3] != 2 {
		t.Errorf("remap[3]: got %d, want 2", remap[3])
	}
	// glyph 4 moves to 3
	if remap[4] != 3 {
		t.Errorf("remap[4]: got %d, want 3", remap[4])
	}
	// glyph 5 was removed
	if _, ok := remap[5]; ok {
		t.Error("remap should not contain removed index 5")
	}
	// glyph 6 moves to 4
	if remap[6] != 4 {
		t.Errorf("remap[6]: got %d, want 4", remap[6])
	}

	// Check maxp updated
	if ttf.maxp.numGlyphs != uint16(origCount-2) {
		t.Errorf("maxp.numGlyphs: got %d, want %d", ttf.maxp.numGlyphs, origCount-2)
	}

	// Check rune mapping updated
	for r, gid := range ttf.runeToGlyphID {
		oldGID := gid
		_ = oldGID
		if int(gid) >= ttf.NumGlyphs() {
			t.Errorf("rune 0x%X maps to glyph %d but only %d glyphs", r, gid, ttf.NumGlyphs())
		}
	}

	// Cannot remove glyph 0
	_, err = ttf.RemoveGlyphs([]int{0})
	if err == nil {
		t.Error("expected error when removing glyph 0")
	}
}

func TestRemoveGlyphsSerialize(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Remove some glyphs
	_, err = ttf.RemoveGlyphs([]int{3, 7, 10})
	if err != nil {
		t.Fatal(err)
	}

	// Serialize and re-parse
	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse serialized font after remove: %v", err)
	}

	// Verify glyph count
	if ttf2.NumGlyphs() != ttf.NumGlyphs() {
		t.Errorf("glyph count mismatch after round-trip: got %d, want %d", ttf2.NumGlyphs(), ttf.NumGlyphs())
	}

	// Verify maxp
	if ttf2.maxp.numGlyphs != ttf.maxp.numGlyphs {
		t.Errorf("maxp.numGlyphs mismatch: got %d, want %d", ttf2.maxp.numGlyphs, ttf.maxp.numGlyphs)
	}

	// Verify cmap mapping still works (for remaining glyphs)
	for _, sub := range ttf2.cmap.subtables {
		if sub.Format() == 4 {
			// 0xE001 should still map to glyph 1 (not removed)
			if gid := sub.Map(0xE001); gid != 1 {
				t.Errorf("Map(0xE001) after round-trip: got %d, want 1", gid)
			}
		}
	}
}

func TestSetRuneMappingSerialize(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Add a mapping for 'A' to glyph 1
	err = ttf.SetRuneMapping(0x41, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize and re-parse
	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse serialized font: %v", err)
	}

	// Check that the mapping survived the round-trip
	if gid := ttf2.RuneToGlyphID(0x41); gid != 1 {
		t.Errorf("RuneToGlyphID('A') after round-trip: got %d, want 1", gid)
	}

	// Original mapping should still work
	if gid := ttf2.RuneToGlyphID(0xE001); gid != 1 {
		t.Errorf("RuneToGlyphID(0xE001) after round-trip: got %d, want 1", gid)
	}
}

func TestRuneMappings(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	mappings := ttf.RuneMappings()
	if len(mappings) == 0 {
		t.Fatal("RuneMappings returned empty")
	}

	// Check sorted order
	for i := 1; i < len(mappings); i++ {
		if mappings[i].Rune <= mappings[i-1].Rune {
			t.Errorf("mappings not sorted: [%d]=0x%X <= [%d]=0x%X", i-1, mappings[i-1].Rune, i, mappings[i].Rune)
		}
	}
}

func TestEnumerateFormat4(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	for _, sub := range ttf.cmap.subtables {
		if sub.Format() == 4 {
			count := 0
			sub.Enumerate(func(r rune, gid uint16) {
				// Verify Enumerate matches Map
				if sub.Map(r) != gid {
					t.Errorf("Enumerate/Map mismatch: rune 0x%X Enumerate=%d Map=%d", r, gid, sub.Map(r))
				}
				count++
			})

			// Test font has glyphs for 0xE001-0xE02F (0xE001 through 0xE030 = 48 entries)
			// But some may map to glyph 0 so Enumerate skips them
			if count == 0 {
				t.Error("Format 4 Enumerate returned 0 entries")
			}
			t.Logf("Format 4 Enumerate: %d entries", count)
		}
	}
}

func TestRemoveRuneMappingSerialize(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	// Remove mapping for 0xE001
	ttf.RemoveRuneMapping(0xE001)

	// Verify it's gone from the abstract map
	if gid := ttf.RuneToGlyphID(0xE001); gid != 0 {
		t.Errorf("RuneToGlyphID(0xE001) after remove: got %d, want 0", gid)
	}

	// Serialize and re-parse
	serialized, err := ttf.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	ttf2, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Mapping should be gone in the serialized font
	if gid := ttf2.RuneToGlyphID(0xE001); gid != 0 {
		t.Errorf("RuneToGlyphID(0xE001) after round-trip: got %d, want 0", gid)
	}

	// Other mappings should still work
	if gid := ttf2.RuneToGlyphID(0xE002); gid != 2 {
		t.Errorf("RuneToGlyphID(0xE002) after round-trip: got %d, want 2", gid)
	}
}
