package gofont

import "testing"

func TestParseGsub(t *testing.T) {
	ttf, err := Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}

	if ttf.gsub == nil {
		t.Fatal("GSUB table is nil")
	}

	if ttf.gsub.MajorVersion != 1 {
		t.Errorf("GSUB major version: got %d, want 1", ttf.gsub.MajorVersion)
	}
	if ttf.gsub.MinorVersion != 0 {
		t.Errorf("GSUB minor version: got %d, want 0", ttf.gsub.MinorVersion)
	}

	// Check LookupList
	if len(ttf.gsub.LookupList.Lookups) != 5 {
		t.Errorf("GSUB lookup count: got %d, want 5", len(ttf.gsub.LookupList.Lookups))
	}

	// Check lookup types: 6, 2, 6, 1, 1
	expectedTypes := []uint16{6, 2, 6, 1, 1}
	for i, lk := range ttf.gsub.LookupList.Lookups {
		if i < len(expectedTypes) && lk.LookupType != expectedTypes[i] {
			t.Errorf("GSUB lookup %d type: got %d, want %d", i, lk.LookupType, expectedTypes[i])
		}
	}
}

func TestRoundTripGsub(t *testing.T) {
	ttf, err := Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}

	written := writeGsub(ttf.gsub)
	gsub2, err := parseGsub(written)
	if err != nil {
		t.Fatal(err)
	}

	if gsub2.MajorVersion != ttf.gsub.MajorVersion {
		t.Errorf("major version mismatch")
	}
	if gsub2.MinorVersion != ttf.gsub.MinorVersion {
		t.Errorf("minor version mismatch")
	}
	if len(gsub2.LookupList.Lookups) != len(ttf.gsub.LookupList.Lookups) {
		t.Errorf("lookup count mismatch: %d vs %d", len(gsub2.LookupList.Lookups), len(ttf.gsub.LookupList.Lookups))
	}

	for i, lk1 := range ttf.gsub.LookupList.Lookups {
		if i >= len(gsub2.LookupList.Lookups) {
			break
		}
		lk2 := gsub2.LookupList.Lookups[i]
		if lk1.LookupType != lk2.LookupType {
			t.Errorf("lookup %d type mismatch: %d vs %d", i, lk1.LookupType, lk2.LookupType)
		}
		if lk1.LookupFlag != lk2.LookupFlag {
			t.Errorf("lookup %d flag mismatch", i)
		}
		if len(lk1.SubTables) != len(lk2.SubTables) {
			t.Errorf("lookup %d subtable count mismatch: %d vs %d", i, len(lk1.SubTables), len(lk2.SubTables))
			continue
		}
		for j, st1 := range lk1.SubTables {
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
