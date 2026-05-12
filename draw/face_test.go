package draw_test

import (
	"fmt"
	"image"
	"image/color"
	imdraw "image/draw"
	"os"
	"path/filepath"
	"sync"
	"testing"

	gofont "github.com/venusliang/go-font"
	fontdraw "github.com/venusliang/go-font/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var (
	testFontData     []byte
	testFontDataOnce sync.Once

	kernFontData     []byte
	kernFontDataOnce sync.Once
)

func loadTestFont(t *testing.T) []byte {
	t.Helper()
	testFontDataOnce.Do(func() {
		data, err := os.ReadFile(filepath.Join("..", "testdata", "Microsoft-Yahei.ttf"))
		if err != nil {
			t.Fatalf("failed to load test font: %v", err)
		}
		testFontData = data
	})
	return testFontData
}

func loadKernFont(t *testing.T) []byte {
	t.Helper()
	kernFontDataOnce.Do(func() {
		data, err := os.ReadFile(filepath.Join("..", "testdata", "LEELAWDB.TTF"))
		if err != nil {
			t.Fatalf("failed to load kern font: %v", err)
		}
		kernFontData = data
	})
	return kernFontData
}

func parseTestFont(t *testing.T) *gofont.TrueTypeFont {
	t.Helper()
	ttf, err := gofont.Parse(loadTestFont(t))
	if err != nil {
		t.Fatal(err)
	}
	return &ttf
}

func parseKernTestFont(t *testing.T) *gofont.TrueTypeFont {
	t.Helper()
	ttf, err := gofont.Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}
	return &ttf
}

func TestNewFaceDefaults(t *testing.T) {
	ttf := parseTestFont(t)
	face := fontdraw.NewFace(ttf, nil)
	if face == nil {
		t.Fatal("NewFace returned nil")
	}
	face.Close()
}

func TestFaceMetrics(t *testing.T) {
	ttf := parseTestFont(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	m := face.Metrics()

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

	expectedAscent := fixed.Int26_6(2167) * 768 / 2048
	if m.Ascent != expectedAscent {
		t.Errorf("Ascent: got %d, want %d", m.Ascent, expectedAscent)
	}
	expectedDescent := fixed.Int26_6(536) * 768 / 2048
	if m.Descent != expectedDescent {
		t.Errorf("Descent: got %d, want %d", m.Descent, expectedDescent)
	}
}

func TestGlyphAdvance(t *testing.T) {
	ttf := parseTestFont(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	advance, ok := face.GlyphAdvance(0x0000)
	if ok {
		t.Error("expected ok=false for unmapped rune")
	}
	if advance != 0 {
		t.Errorf("expected advance=0 for unmapped rune, got %d", advance)
	}

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
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	_, _, ok := face.GlyphBounds(0x0000)
	if ok {
		t.Error("expected ok=false for unmapped rune")
	}

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
	_ = bounds
}

func TestKern(t *testing.T) {
	ttf := parseKernTestFont(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

	k := face.Kern('A', 'V')
	_ = k

	k = face.Kern(0x0000, 0x0001)
	if k != 0 {
		t.Errorf("Kern for unmapped runes should be 0, got %d", k)
	}
}

func TestGlyphRendering(t *testing.T) {
	ttf := parseTestFont(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 24, DPI: 72})
	defer face.Close()

	runes := ttf.MappedRunes()
	if len(runes) == 0 {
		t.Fatal("test font has no mapped runes")
	}

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
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 16, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 200, 50))
	imdraw.Draw(img, img.Bounds(), image.White, image.Point{}, imdraw.Src)

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

	s := ""
	for i := 0; i < 5 && i < len(runes); i++ {
		s += string(runes[i])
	}
	d.DrawString(s)

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
		face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: size, DPI: 72})
		m := face.Metrics()
		if m.Ascent == 0 {
			t.Errorf("Size %.0f: Ascent is zero", size)
		}
		face.Close()
	}
}

func TestFaceEmptyGlyph(t *testing.T) {
	ttf := parseTestFont(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 12, DPI: 72})
	defer face.Close()

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
	_ = dr
	_ = mask
}

// --- font.Drawer integration tests using LEELAWDB.TTF ---

func loadKernFontData(t *testing.T) *gofont.TrueTypeFont {
	t.Helper()
	ttf, err := gofont.Parse(loadKernFont(t))
	if err != nil {
		t.Fatal(err)
	}
	return &ttf
}

func TestDrawStringBasic(t *testing.T) {
	ttf := loadKernFontData(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 32, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 300, 60))
	imdraw.Draw(img, img.Bounds(), image.White, image.Point{}, imdraw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(10, 40),
	}
	d.DrawString("Hello World")

	if !hasNonWhitePixels(img) {
		t.Error("DrawString(\"Hello World\") produced no visible output")
	}
}

func TestDrawStringSingleChar(t *testing.T) {
	ttf := loadKernFontData(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 48, DPI: 72})
	defer face.Close()

	for _, ch := range "ABC" {
		img := image.NewRGBA(image.Rect(0, 0, 100, 80))
		imdraw.Draw(img, img.Bounds(), image.White, image.Point{}, imdraw.Src)

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
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 32, DPI: 72})
	defer face.Close()

	advAV := measureString(face, "AV")
	advAZ := measureString(face, "AZ")

	if advAV >= advAZ {
		t.Errorf("\"AV\" should be narrower than \"AZ\" (kern applies), got AV=%d AZ=%d", advAV, advAZ)
	}
}

func TestDrawStringMultipleLines(t *testing.T) {
	ttf := loadKernFontData(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 20, DPI: 72})
	defer face.Close()

	m := face.Metrics()
	lineHeight := m.Height >> 6

	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	imdraw.Draw(img, img.Bounds(), image.White, image.Point{}, imdraw.Src)

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

	if !hasNonWhitePixels(img) {
		t.Error("multi-line DrawString produced no visible output")
	}
}

func TestDrawStringColoredText(t *testing.T) {
	ttf := loadKernFontData(t)
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 24, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 200, 50))
	imdraw.Draw(img, img.Bounds(), image.White, image.Point{}, imdraw.Src)

	blue := image.NewUniform(color.RGBA{0, 0, 255, 255})
	d := &font.Drawer{
		Dst:  img,
		Src:  blue,
		Face: face,
		Dot:  fixed.P(10, 35),
	}
	d.DrawString("Blue")

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
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 24, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 400, 50))
	imdraw.Draw(img, img.Bounds(), image.White, image.Point{}, imdraw.Src)

	startX := fixed.I(10)
	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(10, 35),
	}
	d.DrawString("ABCD")

	endX := d.Dot.X
	if endX <= startX {
		t.Errorf("Dot.X did not advance: start=%d end=%d", startX, endX)
	}

	advance := (endX - startX) >> 6
	if advance < 20 || advance > 200 {
		t.Errorf("unexpected total advance for \"ABCD\": %d pixels", advance)
	}
}

func TestDrawStringAtDifferentSizes(t *testing.T) {
	ttf := loadKernFontData(t)

	sizes := []float64{10, 16, 32, 64}
	for _, size := range sizes {
		t.Run(formatSize(size), func(t *testing.T) {
			face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: size, DPI: 72})
			defer face.Close()

			m := face.Metrics()
			imgH := int(m.Height>>6) + 10
			if imgH < 20 {
				imgH = 20
			}

			img := image.NewRGBA(image.Rect(0, 0, 200, imgH))
			imdraw.Draw(img, img.Bounds(), image.White, image.Point{}, imdraw.Src)

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
	face := fontdraw.NewFace(ttf, &fontdraw.FaceOptions{Size: 24, DPI: 72})
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, 300, 50))
	imdraw.Draw(img, img.Bounds(), image.White, image.Point{}, imdraw.Src)

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

func formatSize(size float64) string {
	return fmt.Sprintf("Size%.0f", size)
}

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

func measureString(face *fontdraw.Face, s string) fixed.Int26_6 {
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
