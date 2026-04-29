package gofont

import "testing"

func TestParseGpos(t *testing.T) {
	ttf, err := Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}

	if ttf.gpos == nil {
		t.Fatal("GPOS table is nil")
	}

	if ttf.gpos.MajorVersion != 1 {
		t.Errorf("GPOS major version: got %d, want 1", ttf.gpos.MajorVersion)
	}
	if ttf.gpos.MinorVersion != 0 {
		t.Errorf("GPOS minor version: got %d, want 0", ttf.gpos.MinorVersion)
	}

	// Check ScriptList
	if len(ttf.gpos.ScriptList.Records) == 0 {
		t.Error("GPOS ScriptList is empty")
	}

	// Check FeatureList
	if len(ttf.gpos.FeatureList.Records) == 0 {
		t.Error("GPOS FeatureList is empty")
	}

	// Check LookupList
	if len(ttf.gpos.LookupList.Lookups) != 16 {
		t.Errorf("GPOS lookup count: got %d, want 16", len(ttf.gpos.LookupList.Lookups))
	}

	// Check lookup types
	expectedTypes := []uint16{8, 2, 1, 4, 6, 6, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	for i, lk := range ttf.gpos.LookupList.Lookups {
		if i < len(expectedTypes) && lk.LookupType != expectedTypes[i] {
			t.Errorf("GPOS lookup %d type: got %d, want %d", i, lk.LookupType, expectedTypes[i])
		}
	}
}

func TestRoundTripGpos(t *testing.T) {
	ttf, err := Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}

	written := writeGpos(ttf.gpos)
	gpos2, err := parseGpos(written)
	if err != nil {
		t.Fatal(err)
	}

	if gpos2.MajorVersion != ttf.gpos.MajorVersion {
		t.Errorf("major version mismatch")
	}
	if gpos2.MinorVersion != ttf.gpos.MinorVersion {
		t.Errorf("minor version mismatch")
	}
	if len(gpos2.ScriptList.Records) != len(ttf.gpos.ScriptList.Records) {
		t.Errorf("script records mismatch: %d vs %d", len(gpos2.ScriptList.Records), len(ttf.gpos.ScriptList.Records))
	}
	if len(gpos2.FeatureList.Records) != len(ttf.gpos.FeatureList.Records) {
		t.Errorf("feature records mismatch")
	}
	if len(gpos2.LookupList.Lookups) != len(ttf.gpos.LookupList.Lookups) {
		t.Errorf("lookup count mismatch: %d vs %d", len(gpos2.LookupList.Lookups), len(ttf.gpos.LookupList.Lookups))
	}

	// Verify lookup types and subtable counts match
	for i, lk1 := range ttf.gpos.LookupList.Lookups {
		if i >= len(gpos2.LookupList.Lookups) {
			break
		}
		lk2 := gpos2.LookupList.Lookups[i]
		if lk1.LookupType != lk2.LookupType {
			t.Errorf("lookup %d type mismatch: %d vs %d", i, lk1.LookupType, lk2.LookupType)
		}
		if lk1.LookupFlag != lk2.LookupFlag {
			t.Errorf("lookup %d flag mismatch: 0x%04x vs 0x%04x", i, lk1.LookupFlag, lk2.LookupFlag)
		}
		if len(lk1.SubTables) != len(lk2.SubTables) {
			t.Errorf("lookup %d subtable count mismatch: %d vs %d", i, len(lk1.SubTables), len(lk2.SubTables))
		}
		// Verify subtable bytes match
		for j, st1 := range lk1.SubTables {
			if j >= len(lk2.SubTables) {
				break
			}
			st2 := lk2.SubTables[j]
			if len(st1) != len(st2) {
				t.Errorf("lookup %d subtable %d size mismatch: %d vs %d", i, j, len(st1), len(st2))
			} else {
				for k := range st1 {
					if st1[k] != st2[k] {
						t.Errorf("lookup %d subtable %d byte %d mismatch", i, j, k)
						break
					}
				}
			}
		}
	}
}
