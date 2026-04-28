package gofonts

import (
	"testing"
)

func getPost(t *testing.T) *Post {
	t.Helper()
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	if ttf.post == nil {
		t.Fatal("post table is nil")
	}
	return ttf.post
}

func TestParsePost(t *testing.T) {
	post := getPost(t)

	if post.version != 0x00020000 {
		t.Errorf("version: got 0x%08X, want 0x00020000", post.version)
	}
	if post.italicAngle != 0 {
		t.Errorf("italicAngle: got %d, want 0", post.italicAngle)
	}
	if post.underlinePosition != 50 {
		t.Errorf("underlinePosition: got %d, want 50", post.underlinePosition)
	}
	if post.underlineThickness != 0 {
		t.Errorf("underlineThickness: got %d, want 0", post.underlineThickness)
	}
	if post.numGlyphs != 43 {
		t.Errorf("numGlyphs: got %d, want 43", post.numGlyphs)
	}
	if len(post.glyphNameIndex) != 43 {
		t.Fatalf("glyphNameIndex count: got %d, want 43", len(post.glyphNameIndex))
	}

	// glyph[0] = .notdef (index 0)
	if post.glyphNameIndex[0] != 0 {
		t.Errorf("glyphNameIndex[0]: got %d, want 0", post.glyphNameIndex[0])
	}
	// glyph[1] = uniE001 (index 258, first custom name)
	if post.glyphNameIndex[1] != 258 {
		t.Errorf("glyphNameIndex[1]: got %d, want 258", post.glyphNameIndex[1])
	}
}

func TestRoundTripPost(t *testing.T) {
	post := getPost(t)

	written := writePost(post)
	post2, err := parsePost(written)
	if err != nil {
		t.Fatal(err)
	}

	if post2.version != post.version {
		t.Errorf("version mismatch")
	}
	if post2.italicAngle != post.italicAngle {
		t.Errorf("italicAngle mismatch")
	}
	if post2.underlinePosition != post.underlinePosition {
		t.Errorf("underlinePosition mismatch")
	}
	if post2.underlineThickness != post.underlineThickness {
		t.Errorf("underlineThickness mismatch")
	}
	if post2.numGlyphs != post.numGlyphs {
		t.Errorf("numGlyphs mismatch")
	}
	if len(post2.glyphNameIndex) != len(post.glyphNameIndex) {
		t.Fatalf("glyphNameIndex count mismatch")
	}
	for i, idx := range post.glyphNameIndex {
		if post2.glyphNameIndex[i] != idx {
			t.Errorf("glyphNameIndex[%d] mismatch: got %d, want %d", i, post2.glyphNameIndex[i], idx)
		}
	}
	if string(post2.stringData) != string(post.stringData) {
		t.Errorf("stringData mismatch")
	}
}
