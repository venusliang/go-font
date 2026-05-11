package gofont

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"fmt"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func parseTestFont(t *testing.T) *TrueTypeFont {
	t.Helper()
	ttf, err := Parse(loadFont(t))
	if err != nil {
		t.Fatal(err)
	}
	return &ttf
}

func parseKernTestFont(t *testing.T) *TrueTypeFont {
	t.Helper()
	ttf, err := Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}
	return &ttf
}

func TestNewFaceDefaults(t *testing.T) {
	ttf := parseTestFont(t)
	face := NewFace(ttf, nil)
	if face == nil {
		t.Fatal("NewFace returned nil")
	}
	if face.scale == 0 {
		t.Error("scale is zero")
	}
	face.Close()
}

func TestFaceMetrics(t *testing.T) {
	ttf := parseTestFont(t)
	face := NewFace(ttf, &FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	m := face.Metrics()

	// unitsPerEm=1024, ascent=812, descent=-212, lineGap=92
	// scale = 12 * 72 * 64 / 72 = 768 (26.6 fixed-point)
	// Height = (812 - (-212) + 92) * 768 / 1024 = 1116 * 768 / 1024 = 837
	// Ascent = 812 * 768 / 1024 = 609
	// Descent = 212 * 768 / 1024 = 159
	if m.Height == 0 {
		t.Error("Metrics.Height is zero")
	}
	if m.Ascent == 0 {
		t.Error("Metrics.Ascent is zero")
	}
	if m.Descent == 0 {
		t.Error("Metrics.Descent is zero")
	}
	if m.Ascent <= 0 {
		t.Errorf("Metrics.Ascent should be positive, got %d", m.Ascent)
	}
	if m.Descent < 0 {
		t.Errorf("Metrics.Descent should be non-negative, got %d", m.Descent)
	}

	// Verify expected values
	expectedAscent := fixed.Int26_6(812) * 768 / 1024
	if m.Ascent != expectedAscent {
		t.Errorf("Ascent: got %d, want %d", m.Ascent, expectedAscent)
	}
	expectedDescent := fixed.Int26_6(212) * 768 / 1024
	if m.Descent != expectedDescent {
		t.Errorf("Descent: got %d, want %d", m.Descent, expectedDescent)
	}
}

func TestGlyphAdvance(t *testing.T) {
	ttf := parseTestFont(t)
	face := NewFace(ttf, &FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	// Test unmapped rune
	advance, ok := face.GlyphAdvance(0x0000)
	if ok {
		t.Error("expected ok=false for unmapped rune")
	}
	if advance != 0 {
		t.Errorf("expected advance=0 for unmapped rune, got %d", advance)
	}

	// Test that we can get advance for at least one mapped rune
	runes := ttf.MappedRunes()
	if len(runes) == 0 {
		t.Fatal("test font has no mapped runes")
	}

	r := runes[0]
	gid := ttf.RuneToGlyphID(r)
	if gid == 0 {
		t.Fatal("mapped rune has glyph ID 0")
	}

	advance, ok = face.GlyphAdvance(r)
	if !ok {
		t.Errorf("GlyphAdvance(%q U+%04X, glyphID=%d) returned ok=false", r, r, gid)
	}
	if advance == 0 {
		aw := ttf.AdvanceWidth(gid)
		t.Errorf("GlyphAdvance(%q) returned 0, advanceWidth in design units=%d", r, aw)
	}
}

func TestGlyphBounds(t *testing.T) {
	ttf := parseTestFont(t)
	face := NewFace(ttf, &FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	// Test unmapped rune
	_, _, ok := face.GlyphBounds(0x0000)
	if ok {
		t.Error("expected ok=false for unmapped rune")
	}

	// Test a mapped rune
	runes := ttf.MappedRunes()
	if len(runes) == 0 {
		t.Fatal("test font has no mapped runes")
	}

	bounds, advance, ok := face.GlyphBounds(runes[0])
	if !ok {
		t.Errorf("GlyphBounds(%q) returned ok=false", runes[0])
	}
	if advance == 0 {
		t.Errorf("GlyphBounds(%q) returned zero advance", runes[0])
	}

	// For a glyph with actual outlines, bounds should be non-empty
	// Y-flip: Min.Y should be negative (above baseline) and Max.Y should be positive (below baseline)
	// Actually, after Y-flip: bounds.Min.Y = -scale(yMax), bounds.Max.Y = -scale(yMin)
	// Since yMin < 0 for most glyphs (below baseline) and yMax > 0 (above baseline):
	// Min.Y = -scale(yMax) < 0 (above baseline in image space)
	// Max.Y = -scale(yMin) > 0 (below baseline in image space)
	_ = bounds
}

func TestKern(t *testing.T) {
	// Test with kern font
	ttf := parseKernTestFont(t)
	face := NewFace(ttf, &FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	// Kerning for two runes that have a kern pair
	// Just verify it doesn't crash and returns reasonable values
	k := face.Kern('A', 'V')
	_ = k // kern value may or may not be 0 depending on the font

	// Unmapped runes should return 0
	k = face.Kern(0x0000, 0x0001)
	if k != 0 {
		t.Errorf("Kern for unmapped runes should be 0, got %d", k)
	}
}

func TestGlyphRendering(t *testing.T) {
	ttf := parseTestFont(t)
	face := NewFace(ttf, &FaceOptions{Size: 24, DPI: 72})
	defer face.Close()

	runes := ttf.MappedRunes()
	if len(runes) == 0 {
		t.Fatal("test font has no mapped runes")
	}

	// Try rendering a few glyphs
	rendered := 0
	for _, r := range runes {
		if rendered >= 5 {
			break
		}

		dot := fixed.P(0, int(face.Metrics().Ascent>>6))
		dr, mask, maskp, advance, ok := face.Glyph(dot, r)
		if !ok {
			continue
		}
		rendered++

		_ = maskp
		_ = advance

		if mask == nil {
			// Some glyphs (like space) may have no outline
			if dr.Empty() {
				continue
			}
			t.Errorf("Glyph(%q): non-empty dr but nil mask", r)
		}
	}

	if rendered == 0 {
		t.Error("no glyphs were rendered")
	}
}

func TestFaceDrawString(t *testing.T) {
	ttf := parseTestFont(t)
	face := NewFace(ttf, &FaceOptions{Size: 16, DPI: 72})
	defer face.Close()

	// Draw a string using font.DrawString
	img := image.NewRGBA(image.Rect(0, 0, 200, 50))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	d := font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(10, 30),
	}

	runes := ttf.MappedRunes()
	if len(runes) == 0 {
		t.Fatal("test font has no mapped runes")
	}

	// Draw first 5 mapped characters
	s := ""
	for i := 0; i < 5 && i < len(runes); i++ {
		s += string(runes[i])
	}
	d.DrawString(s)

	// Verify something was drawn (at least one non-white pixel)
	hasContent := false
	for y := 0; y < img.Bounds().Dy() && !hasContent; y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r != 0xFFFF || g != 0xFFFF || b != 0xFFFF {
				hasContent = true
				break
			}
		}
	}
	if !hasContent {
		t.Error("DrawString produced no visible output")
	}
}

func TestFaceDifferentSizes(t *testing.T) {
	ttf := parseTestFont(t)

	sizes := []float64{8, 12, 24, 48, 96}
	for _, size := range sizes {
		face := NewFace(ttf, &FaceOptions{Size: size, DPI: 72})
		m := face.Metrics()
		if m.Ascent == 0 {
			t.Errorf("Size %.0f: Ascent is zero", size)
		}
		face.Close()
	}
}

func TestFaceEmptyGlyph(t *testing.T) {
	ttf := parseTestFont(t)
	face := NewFace(ttf, &FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	// Test space character - typically has advance but no outline
	spaceGid := ttf.RuneToGlyphID(' ')
	if spaceGid == 0 {
		t.Skip("space not mapped in test font")
	}

	dot := fixed.P(0, int(face.Metrics().Ascent>>6))
	dr, mask, _, advance, ok := face.Glyph(dot, ' ')
	if !ok {
		t.Error("Glyph(space) returned ok=false")
	}
	if advance == 0 {
		t.Error("space should have non-zero advance")
	}
	// Space has no outline, so mask should be nil and dr empty
	_ = dr
	_ = mask
}

// --- font.Drawer integration tests using LEELAWDB.TTF ---

func loadKernFontData(t *testing.T) *TrueTypeFont {
	t.Helper()
	ttf, err := Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}
	return &ttf
}

func TestDrawStringBasic(t *testing.T) {
	ttf := loadKernFontData(t)
	face := NewFace(ttf, &FaceOptions{Size: 32, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 300, 60))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(10, 40),
	}
	d.DrawString("Hello World")

	// Verify visible content was drawn
	if !hasNonWhitePixels(img) {
		t.Error("DrawString(\"Hello World\") produced no visible output")
	}
}

func TestDrawStringSingleChar(t *testing.T) {
	ttf := loadKernFontData(t)
	face := NewFace(ttf, &FaceOptions{Size: 48, DPI: 72})
	defer face.Close()

	for _, ch := range "ABC" {
		img := image.NewRGBA(image.Rect(0, 0, 100, 80))
		draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

		d := &font.Drawer{
			Dst:  img,
			Src:  image.Black,
			Face: face,
			Dot:  fixed.P(20, 60),
		}
		d.DrawString(string(ch))

		if !hasNonWhitePixels(img) {
			t.Errorf("DrawString(%q) produced no visible output", ch)
		}
	}
}

func TestDrawStringKerning(t *testing.T) {
	ttf := loadKernFontData(t)
	face := NewFace(ttf, &FaceOptions{Size: 32, DPI: 72})
	defer face.Close()

	// "AV" has a negative kern value, so it should be narrower than "AZ" (no kern)
	advAV := measureString(face, "AV")
	advAZ := measureString(face, "AZ")

	if advAV >= advAZ {
		t.Errorf("\"AV\" should be narrower than \"AZ\" (kern applies), got AV=%d AZ=%d", advAV, advAZ)
	}
}

func TestDrawStringMultipleLines(t *testing.T) {
	ttf := loadKernFontData(t)
	face := NewFace(ttf, &FaceOptions{Size: 20, DPI: 72})
	defer face.Close()

	m := face.Metrics()
	lineHeight := m.Height >> 6

	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
	}

	lines := []string{"First line", "Second line", "Third line"}
	for i, line := range lines {
		d.Dot = fixed.P(10, 20+int(m.Ascent>>6)+i*int(lineHeight))
		d.DrawString(line)
	}

	// Verify content was drawn
	if !hasNonWhitePixels(img) {
		t.Error("multi-line DrawString produced no visible output")
	}
}

func TestDrawStringColoredText(t *testing.T) {
	ttf := loadKernFontData(t)
	face := NewFace(ttf, &FaceOptions{Size: 24, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 200, 50))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	// Draw with a colored source
	blue := image.NewUniform(color.RGBA{0, 0, 255, 255})
	d := &font.Drawer{
		Dst:  img,
		Src:  blue,
		Face: face,
		Dot:  fixed.P(10, 35),
	}
	d.DrawString("Blue")

	// Verify blue pixels exist
	foundBlue := false
	for y := 0; y < img.Bounds().Dy() && !foundBlue; y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if b > 0x8000 && r < 0x4000 && g < 0x4000 {
				foundBlue = true
				break
			}
		}
	}
	if !foundBlue {
		t.Error("colored DrawString produced no blue pixels")
	}
}

func TestDrawStringAdvancesCorrectly(t *testing.T) {
	ttf := loadKernFontData(t)
	face := NewFace(ttf, &FaceOptions{Size: 24, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 400, 50))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	startX := fixed.I(10)
	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(10, 35),
	}
	d.DrawString("ABCD")

	// Verify Dot advanced past the start position
	endX := d.Dot.X
	if endX <= startX {
		t.Errorf("Dot.X did not advance: start=%d end=%d", startX, endX)
	}

	// Verify the advance is reasonable (each char ~10-20px at 24pt)
	advance := (endX - startX) >> 6
	if advance < 20 || advance > 200 {
		t.Errorf("unexpected total advance for \"ABCD\": %d pixels", advance)
	}
}

func TestDrawStringAtDifferentSizes(t *testing.T) {
	ttf := loadKernFontData(t)

	sizes := []float64{10, 16, 32, 64}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size%.0f", size), func(t *testing.T) {
			face := NewFace(ttf, &FaceOptions{Size: size, DPI: 72})
			defer face.Close()

			m := face.Metrics()
			imgH := int(m.Height>>6) + 10
			if imgH < 20 {
				imgH = 20
			}

			img := image.NewRGBA(image.Rect(0, 0, 200, imgH))
			draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

			d := &font.Drawer{
				Dst:  img,
				Src:  image.Black,
				Face: face,
				Dot:  fixed.P(5, int(m.Ascent>>6)+2),
			}
			d.DrawString("Test")

			if !hasNonWhitePixels(img) {
				t.Errorf("size %.0f: DrawString produced no visible output", size)
			}
		})
	}
}

func TestDrawStringWithUnmappedRunes(t *testing.T) {
	ttf := loadKernFontData(t)
	face := NewFace(ttf, &FaceOptions{Size: 24, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 300, 50))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	// String contains a mix of mapped and unmapped characters
	// The string "A\x00B" should draw A and B, skipping the null byte
	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(10, 35),
	}
	d.DrawString("A\x00B")

	if !hasNonWhitePixels(img) {
		t.Error("DrawString with unmapped runes produced no visible output")
	}
}

// --- Helper functions ---

// hasNonWhitePixels checks if the image contains at least one non-white pixel.
func hasNonWhitePixels(img *image.RGBA) bool {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r != 0xFFFF || g != 0xFFFF || b != 0xFFFF {
				return true
			}
		}
	}
	return false
}

// measureString returns the total advance width of a string rendered by the Face.
func measureString(face *Face, s string) fixed.Int26_6 {
	var total fixed.Int26_6
	var prev rune
	for i, r := range s {
		if i > 0 {
			total += face.Kern(prev, r)
		}
		adv, ok := face.GlyphAdvance(r)
		if ok {
			total += adv
		}
		prev = r
	}
	return total
}
