package gofonts

import (
	"testing"
)

func TestParseName(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	name := ttf.name
	if name == nil {
		t.Fatal("name table is nil")
	}

	if name.format != 0 {
		t.Errorf("format: got %d, want 0", name.format)
	}
	if name.count != 12 {
		t.Errorf("count: got %d, want 12", name.count)
	}
	if name.stringOffset != 150 {
		t.Errorf("stringOffset: got %d, want 150", name.stringOffset)
	}
	if len(name.nameRecords) != 12 {
		t.Errorf("nameRecords count: got %d, want 12", len(name.nameRecords))
	}

	// First record: platformID=1, encodingID=0, languageID=0, nameID=1, length=10
	r0 := name.nameRecords[0]
	if r0.platformID != 1 {
		t.Errorf("record[0] platformID: got %d, want 1", r0.platformID)
	}
	if r0.nameID != 1 {
		t.Errorf("record[0] nameID: got %d, want 1", r0.nameID)
	}
	if r0.length != 10 {
		t.Errorf("record[0] length: got %d, want 10", r0.length)
	}

	// Verify string storage contains data
	if len(name.stringStorage) == 0 {
		t.Error("stringStorage is empty")
	}
}

func TestRoundTripName(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	written := writeName(ttf.name)
	name2, err := parseName(written)
	if err != nil {
		t.Fatal(err)
	}

	orig := ttf.name
	if name2.format != orig.format {
		t.Errorf("format mismatch: %d vs %d", name2.format, orig.format)
	}
	if name2.count != orig.count {
		t.Errorf("count mismatch: %d vs %d", name2.count, orig.count)
	}
	if name2.stringOffset != orig.stringOffset {
		t.Errorf("stringOffset mismatch: %d vs %d", name2.stringOffset, orig.stringOffset)
	}
	if len(name2.nameRecords) != len(orig.nameRecords) {
		t.Fatalf("nameRecords count mismatch: %d vs %d", len(name2.nameRecords), len(orig.nameRecords))
	}
	for i, r := range orig.nameRecords {
		r2 := name2.nameRecords[i]
		if r2 != r {
			t.Errorf("nameRecord[%d] mismatch: got %+v, want %+v", i, r2, r)
		}
	}
	if string(name2.stringStorage) != string(orig.stringStorage) {
		t.Errorf("stringStorage mismatch")
	}
}
