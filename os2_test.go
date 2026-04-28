package gofont

import (
	"testing"
)

func TestParseOS2(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	os2 := ttf.os2
	if os2 == nil {
		t.Fatal("os2 table is nil")
	}

	if os2.version != 4 {
		t.Errorf("version: got %d, want 4", os2.version)
	}
	if os2.xAvgCharWidth != 942 {
		t.Errorf("xAvgCharWidth: got %d, want 942", os2.xAvgCharWidth)
	}
	if os2.usWeightClass != 400 {
		t.Errorf("usWeightClass: got %d, want 400", os2.usWeightClass)
	}
	if os2.usWidthClass != 5 {
		t.Errorf("usWidthClass: got %d, want 5", os2.usWidthClass)
	}
	if os2.fsType != 0 {
		t.Errorf("fsType: got %d, want 0", os2.fsType)
	}
	if os2.sFamilyClass != 0 {
		t.Errorf("sFamilyClass: got %d, want 0", os2.sFamilyClass)
	}
	expectedPanose := [10]byte{2, 0, 6, 3, 0, 0, 0, 0, 0, 0}
	if os2.panose != expectedPanose {
		t.Errorf("panose: got %v, want %v", os2.panose, expectedPanose)
	}
	if os2.achVendID != [4]byte{'P', 'f', 'E', 'd'} {
		t.Errorf("achVendID: got %q, want 'PfEd'", string(os2.achVendID[:]))
	}
	if os2.fsSelection != 0x00C0 {
		t.Errorf("fsSelection: got 0x%04X, want 0x00C0", os2.fsSelection)
	}
	if os2.usFirstCharIndex != 0xE001 {
		t.Errorf("usFirstCharIndex: got 0x%04X, want 0xE001", os2.usFirstCharIndex)
	}
	if os2.usLastCharIndex != 0xE030 {
		t.Errorf("usLastCharIndex: got 0x%04X, want 0xE030", os2.usLastCharIndex)
	}
	if os2.sTypoAscender != 812 {
		t.Errorf("sTypoAscender: got %d, want 812", os2.sTypoAscender)
	}
	if os2.sTypoDescender != -212 {
		t.Errorf("sTypoDescender: got %d, want -212", os2.sTypoDescender)
	}
	if os2.sTypoLineGap != 92 {
		t.Errorf("sTypoLineGap: got %d, want 92", os2.sTypoLineGap)
	}
	if os2.usWinAscent != 812 {
		t.Errorf("usWinAscent: got %d, want 812", os2.usWinAscent)
	}
	if os2.usWinDescent != 212 {
		t.Errorf("usWinDescent: got %d, want 212", os2.usWinDescent)
	}
	if os2.ulCodePageRange1 != 0x00000001 {
		t.Errorf("ulCodePageRange1: got 0x%08X, want 0x00000001", os2.ulCodePageRange1)
	}
	if os2.ulCodePageRange2 != 0x00000000 {
		t.Errorf("ulCodePageRange2: got 0x%08X, want 0x00000000", os2.ulCodePageRange2)
	}
	if os2.sxHeight != 792 {
		t.Errorf("sxHeight: got %d, want 792", os2.sxHeight)
	}
	if os2.usBreakChar != 32 {
		t.Errorf("usBreakChar: got %d, want 32", os2.usBreakChar)
	}
	if os2.usMaxContext != 1 {
		t.Errorf("usMaxContext: got %d, want 1", os2.usMaxContext)
	}
}

func TestRoundTripOS2(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	written := writeOS2(ttf.os2)
	os2v2, err := parseOS2(written)
	if err != nil {
		t.Fatal(err)
	}

	orig := ttf.os2
	if os2v2.version != orig.version {
		t.Errorf("version mismatch: %d vs %d", os2v2.version, orig.version)
	}
	if os2v2.xAvgCharWidth != orig.xAvgCharWidth {
		t.Errorf("xAvgCharWidth mismatch")
	}
	if os2v2.usWeightClass != orig.usWeightClass {
		t.Errorf("usWeightClass mismatch")
	}
	if os2v2.usWidthClass != orig.usWidthClass {
		t.Errorf("usWidthClass mismatch")
	}
	if os2v2.panose != orig.panose {
		t.Errorf("panose mismatch")
	}
	if os2v2.achVendID != orig.achVendID {
		t.Errorf("achVendID mismatch")
	}
	if os2v2.fsSelection != orig.fsSelection {
		t.Errorf("fsSelection mismatch")
	}
	if os2v2.sTypoAscender != orig.sTypoAscender {
		t.Errorf("sTypoAscender mismatch")
	}
	if os2v2.sTypoDescender != orig.sTypoDescender {
		t.Errorf("sTypoDescender mismatch")
	}
	if os2v2.sTypoLineGap != orig.sTypoLineGap {
		t.Errorf("sTypoLineGap mismatch")
	}
	if os2v2.ulCodePageRange1 != orig.ulCodePageRange1 {
		t.Errorf("ulCodePageRange1 mismatch")
	}
	if os2v2.ulCodePageRange2 != orig.ulCodePageRange2 {
		t.Errorf("ulCodePageRange2 mismatch")
	}
	if os2v2.sxHeight != orig.sxHeight {
		t.Errorf("sxHeight mismatch")
	}
	if os2v2.sCapHeight != orig.sCapHeight {
		t.Errorf("sCapHeight mismatch")
	}
	if os2v2.usDefaultChar != orig.usDefaultChar {
		t.Errorf("usDefaultChar mismatch")
	}
	if os2v2.usBreakChar != orig.usBreakChar {
		t.Errorf("usBreakChar mismatch")
	}
	if os2v2.usMaxContext != orig.usMaxContext {
		t.Errorf("usMaxContext mismatch")
	}
}
