package gofont

import (
	"testing"
)

func TestParsePrep(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.prep == nil {
		t.Fatal("prep table is nil")
	}
	if len(ttf.prep.instructions) == 0 {
		t.Error("prep instructions is empty")
	}
}

func TestParseCvt(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.cvt == nil {
		t.Skip("cvt table not present in test font")
	}
	if len(ttf.cvt.values) == 0 {
		t.Error("cvt values is empty")
	}
}

func TestParseFpgm(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.fpgm == nil {
		t.Fatal("fpgm table is nil")
	}
	if len(ttf.fpgm.instructions) == 0 {
		t.Error("fpgm instructions is empty")
	}
}

func TestRoundTripPrep(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	data := writePrep(ttf.prep)
	prep2 := parsePrep(data)

	if len(prep2.instructions) != len(ttf.prep.instructions) {
		t.Fatalf("prep instructions length mismatch: %d vs %d", len(prep2.instructions), len(ttf.prep.instructions))
	}
	for i, b := range ttf.prep.instructions {
		if prep2.instructions[i] != b {
			t.Errorf("prep byte mismatch at %d: %02x vs %02x", i, prep2.instructions[i], b)
		}
	}
}

func TestRoundTripCvt(t *testing.T) {
	// Test with synthetic data since test font may not have cvt
	original := &Cvt{values: []int16{100, -200, 0, 500, -50}}
	data := writeCvt(original)
	cvt2 := parseCvt(data)

	if len(cvt2.values) != len(original.values) {
		t.Fatalf("cvt values length mismatch: %d vs %d", len(cvt2.values), len(original.values))
	}
	for i, v := range original.values {
		if cvt2.values[i] != v {
			t.Errorf("cvt value mismatch at %d: %d vs %d", i, cvt2.values[i], v)
		}
	}
}

func TestRoundTripFpgm(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	data := writeFpgm(ttf.fpgm)
	fpgm2 := parseFpgm(data)

	if len(fpgm2.instructions) != len(ttf.fpgm.instructions) {
		t.Fatalf("fpgm instructions length mismatch: %d vs %d", len(fpgm2.instructions), len(ttf.fpgm.instructions))
	}
	for i, b := range ttf.fpgm.instructions {
		if fpgm2.instructions[i] != b {
			t.Errorf("fpgm byte mismatch at %d: %02x vs %02x", i, fpgm2.instructions[i], b)
		}
	}
}
