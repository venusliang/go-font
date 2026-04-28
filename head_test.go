package gofonts

import (
	"testing"
)

func TestParseHead(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	head := ttf.head
	if head == nil {
		t.Fatal("head table is nil")
	}

	if head.majorVersion != 1 {
		t.Errorf("majorVersion: got %d, want 1", head.majorVersion)
	}
	if head.minorVersion != 0 {
		t.Errorf("minorVersion: got %d, want 0", head.minorVersion)
	}
	if head.magicNumber != 0x5F0F3CF5 {
		t.Errorf("magicNumber: got 0x%08X, want 0x5F0F3CF5", head.magicNumber)
	}
	if head.unitsPerEm != 1024 {
		t.Errorf("unitsPerEm: got %d, want 1024", head.unitsPerEm)
	}
	if head.flags != 0x000B {
		t.Errorf("flags: got 0x%04X, want 0x000B", head.flags)
	}
	if head.indexToLocFormat != 0 {
		t.Errorf("indexToLocFormat: got %d, want 0 (short)", head.indexToLocFormat)
	}
	if head.glyphDataFormat != 0 {
		t.Errorf("glyphDataFormat: got %d, want 0", head.glyphDataFormat)
	}
	if head.xMin != 6 {
		t.Errorf("xMin: got %d, want 6", head.xMin)
	}
	if head.yMin != -206 {
		t.Errorf("yMin: got %d, want -206", head.yMin)
	}
	if head.xMax != 1321 {
		t.Errorf("xMax: got %d, want 1321", head.xMax)
	}
	if head.yMax != 808 {
		t.Errorf("yMax: got %d, want 808", head.yMax)
	}
	if head.macStyle != 0 {
		t.Errorf("macStyle: got 0x%04X, want 0", head.macStyle)
	}
	if head.lowestRecPPEM != 8 {
		t.Errorf("lowestRecPPEM: got %d, want 8", head.lowestRecPPEM)
	}
	if head.fontDirectionHint != 2 {
		t.Errorf("fontDirectionHint: got %d, want 2", head.fontDirectionHint)
	}
}

func TestRoundTripHead(t *testing.T) {
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}

	written := writeHead(ttf.head)
	head2, err := parseHead(written)
	if err != nil {
		t.Fatal(err)
	}

	orig := ttf.head
	if head2.majorVersion != orig.majorVersion {
		t.Errorf("majorVersion mismatch: %d vs %d", head2.majorVersion, orig.majorVersion)
	}
	if head2.minorVersion != orig.minorVersion {
		t.Errorf("minorVersion mismatch")
	}
	if head2.fontRevision != orig.fontRevision {
		t.Errorf("fontRevision mismatch")
	}
	if head2.magicNumber != orig.magicNumber {
		t.Errorf("magicNumber mismatch")
	}
	if head2.flags != orig.flags {
		t.Errorf("flags mismatch")
	}
	if head2.unitsPerEm != orig.unitsPerEm {
		t.Errorf("unitsPerEm mismatch")
	}
	if head2.created != orig.created {
		t.Errorf("created mismatch")
	}
	if head2.modified != orig.modified {
		t.Errorf("modified mismatch")
	}
	if head2.xMin != orig.xMin || head2.yMin != orig.yMin {
		t.Errorf("min bounds mismatch")
	}
	if head2.xMax != orig.xMax || head2.yMax != orig.yMax {
		t.Errorf("max bounds mismatch")
	}
	if head2.macStyle != orig.macStyle {
		t.Errorf("macStyle mismatch")
	}
	if head2.lowestRecPPEM != orig.lowestRecPPEM {
		t.Errorf("lowestRecPPEM mismatch")
	}
	if head2.fontDirectionHint != orig.fontDirectionHint {
		t.Errorf("fontDirectionHint mismatch")
	}
	if head2.indexToLocFormat != orig.indexToLocFormat {
		t.Errorf("indexToLocFormat mismatch")
	}
	if head2.glyphDataFormat != orig.glyphDataFormat {
		t.Errorf("glyphDataFormat mismatch")
	}
}
