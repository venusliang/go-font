package gofont

import (
	"testing"
)

func TestParseKern(t *testing.T) {
	ttf, err := Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}

	if ttf.kern == nil {
		t.Fatal("kern table is nil")
	}

	if ttf.kern.version != 0 {
		t.Errorf("kern version: got %d, want 0", ttf.kern.version)
	}
	if len(ttf.kern.subtables) != 1 {
		t.Fatalf("kern subtables: got %d, want 1", len(ttf.kern.subtables))
	}

	sub := ttf.kern.subtables[0]
	if sub.format != 0 {
		t.Fatalf("kern subtable format: got %d, want 0", sub.format)
	}
	if sub.coverage&0x01 == 0 {
		t.Error("kern subtable should be horizontal")
	}
	if len(sub.pairs) != 1448 {
		t.Errorf("kern pairs count: got %d, want 1448", len(sub.pairs))
	}

	// Check first pair
	if len(sub.pairs) > 0 {
		p := sub.pairs[0]
		if p.Left != 5 || p.Right != 85 || p.Value != -41 {
			t.Errorf("first kern pair: got (%d, %d, %d), want (5, 85, -41)", p.Left, p.Right, p.Value)
		}
	}
}

func TestRoundTripKern(t *testing.T) {
	ttf, err := Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}

	written := writeKern(ttf.kern)
	kern2, err := parseKern(written)
	if err != nil {
		t.Fatal(err)
	}

	if kern2.version != ttf.kern.version {
		t.Errorf("version mismatch")
	}
	if len(kern2.subtables) != len(ttf.kern.subtables) {
		t.Fatalf("subtable count mismatch: %d vs %d", len(kern2.subtables), len(ttf.kern.subtables))
	}
	for i, sub1 := range ttf.kern.subtables {
		sub2 := kern2.subtables[i]
		if sub1.format != sub2.format {
			t.Errorf("subtable %d format mismatch", i)
		}
		if sub1.coverage != sub2.coverage {
			t.Errorf("subtable %d coverage mismatch: 0x%04x vs 0x%04x", i, sub1.coverage, sub2.coverage)
		}
		if len(sub1.pairs) != len(sub2.pairs) {
			t.Fatalf("subtable %d pair count mismatch: %d vs %d", i, len(sub1.pairs), len(sub2.pairs))
		}
		for j, p1 := range sub1.pairs {
			p2 := sub2.pairs[j]
			if p1 != p2 {
				t.Errorf("subtable %d pair %d mismatch: (%d,%d,%d) vs (%d,%d,%d)",
					i, j, p1.Left, p1.Right, p1.Value, p2.Left, p2.Right, p2.Value)
			}
		}
	}
}

// TestKernFontFullRoundTrip verifies parse → serialize → parse of a font with kern/GPOS/GSUB.
func TestKernFontFullRoundTrip(t *testing.T) {
	orig := loadKernFont(t)
	ttf, err := Parse(orig)
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

	// Verify kern round-trip
	if ttf2.kern == nil {
		t.Fatal("kern table lost after serialize")
	}
	if len(ttf2.kern.subtables) != len(ttf.kern.subtables) {
		t.Errorf("kern subtable count mismatch")
	}
	if len(ttf2.kern.subtables) > 0 && len(ttf2.kern.subtables[0].pairs) != len(ttf.kern.subtables[0].pairs) {
		t.Errorf("kern pair count mismatch: %d vs %d", len(ttf2.kern.subtables[0].pairs), len(ttf.kern.subtables[0].pairs))
	}

	// Verify GPOS round-trip
	if ttf2.gpos == nil {
		t.Fatal("GPOS table lost after serialize")
	}
	if len(ttf2.gpos.LookupList.Lookups) != len(ttf.gpos.LookupList.Lookups) {
		t.Errorf("GPOS lookup count mismatch: %d vs %d", len(ttf2.gpos.LookupList.Lookups), len(ttf.gpos.LookupList.Lookups))
	}

	// Verify GSUB round-trip
	if ttf2.gsub == nil {
		t.Fatal("GSUB table lost after serialize")
	}
	if len(ttf2.gsub.LookupList.Lookups) != len(ttf.gsub.LookupList.Lookups) {
		t.Errorf("GSUB lookup count mismatch: %d vs %d", len(ttf2.gsub.LookupList.Lookups), len(ttf.gsub.LookupList.Lookups))
	}
}
