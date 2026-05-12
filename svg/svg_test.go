package svg_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	gofont "github.com/venusliang/go-font"
	svgexp "github.com/venusliang/go-font/svg"
)

var (
	ttfData     []byte
	ttfDataOnce sync.Once

	kernData     []byte
	kernDataOnce sync.Once
)

func loadTTF(t *testing.T) []byte {
	t.Helper()
	ttfDataOnce.Do(func() {
		var err error
		ttfData, err = os.ReadFile(filepath.Join("..", "testdata", "Microsoft-Yahei.ttf"))
		if err != nil {
			t.Fatalf("failed to load test font: %v", err)
		}
	})
	return ttfData
}

func loadKern(t *testing.T) []byte {
	t.Helper()
	kernDataOnce.Do(func() {
		var err error
		kernData, err = os.ReadFile(filepath.Join("..", "testdata", "LEELAWDB.TTF"))
		if err != nil {
			t.Fatalf("failed to load kern font: %v", err)
		}
	})
	return kernData
}

func parseTTF(t *testing.T) *gofont.TrueTypeFont {
	t.Helper()
	f, err := gofont.Parse(loadTTF(t))
	if err != nil {
		t.Fatal(err)
	}
	return &f
}

func parseKern(t *testing.T) *gofont.TrueTypeFont {
	t.Helper()
	f, err := gofont.Parse(loadKern(t))
	if err != nil {
		t.Fatal(err)
	}
	return &f
}

// --- Tests ---

func TestGlyphDefaults(t *testing.T) {
	f := parseTTF(t)

	svg, err := svgexp.Glyph(f, 40, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should be valid SVG with xmlns
	if !strings.Contains(svg, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("missing SVG namespace")
	}

	// Should have viewBox
	if !strings.Contains(svg, `viewBox="`) {
		t.Error("missing viewBox attribute")
	}

	// Should have path with default fill
	if !strings.Contains(svg, `fill="black"`) {
		t.Error("missing default fill=black")
	}

	// Should have path element
	if !strings.Contains(svg, `<path`) {
		t.Error("missing path element")
	}

	// Should close properly
	if !strings.HasSuffix(svg, "</svg>") {
		t.Errorf("bad SVG ending: %q", svg[len(svg)-20:])
	}
}

func TestGlyphCustomOptions(t *testing.T) {
	f := parseTTF(t)

	opts := &svgexp.Options{
		Fill:        "red",
		Stroke:      "blue",
		StrokeWidth: 2.5,
		Padding:     50,
		Scale:       0.5,
	}

	svg, err := svgexp.Glyph(f, 0, opts)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(svg, `fill="red"`) {
		t.Error("missing custom fill")
	}
	if !strings.Contains(svg, `stroke="blue"`) {
		t.Error("missing custom stroke")
	}
	if !strings.Contains(svg, `stroke-width="2.50"`) {
		t.Error("missing custom stroke-width")
	}
}

func TestGlyphZeroPaddingAndScale(t *testing.T) {
	f := parseTTF(t)

	opts := &svgexp.Options{
		Padding: 0,
		Scale:   0,
	}

	svg, err := svgexp.Glyph(f, 0, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Zero padding and scale should use defaults (20 and 1.0)
	if !strings.Contains(svg, `<path`) {
		t.Error("missing path element with zero opts")
	}
}

func TestGlyphForRune(t *testing.T) {
	f := parseTTF(t)

	svg, err := svgexp.GlyphForRune(f, 'A', nil)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(svg, `<path`) {
		t.Error("GlyphForRune('A') should produce a valid SVG")
	}
}

func TestGlyphForRuneUnmapped(t *testing.T) {
	f := parseTTF(t)

	_, err := svgexp.GlyphForRune(f, '\x00', nil)
	if err == nil {
		t.Error("expected error for unmapped rune U+0000")
	}
}

func TestGlyphInvalidIndex(t *testing.T) {
	f := parseTTF(t)

	_, err := svgexp.Glyph(f, 999999, nil)
	if err == nil {
		t.Error("expected error for out-of-range glyph index")
	}

	_, err = svgexp.Glyph(f, -1, nil)
	if err == nil {
		t.Error("expected error for negative glyph index")
	}
}

func TestGlyphContoursClosed(t *testing.T) {
	f := parseTTF(t)

	svg, err := svgexp.Glyph(f, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Path data should end with Z (closed contour)
	// Extract the d attribute value
	dStart := strings.Index(svg, `d="`)
	if dStart < 0 {
		t.Fatal("missing d attribute")
	}
	dEnd := strings.Index(svg[dStart+3:], `"`)
	if dEnd < 0 {
		t.Fatal("unterminated d attribute")
	}
	d := svg[dStart+3 : dStart+3+dEnd]

	if !strings.HasSuffix(strings.TrimSpace(d), "Z") {
		t.Errorf("path data should end with Z: %q", d)
	}
}

func TestGlyphMultipleSizes(t *testing.T) {
	f := parseTTF(t)

	// Use 'A' which we know has a mapped glyph with outlines
	scales := []float64{0.5, 1.0, 2.0, 10.0}
	for _, scale := range scales {
		t.Run(formatScale(scale), func(t *testing.T) {
			svg, err := svgexp.GlyphForRune(f, 'A', &svgexp.Options{Scale: scale})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(svg, `<path`) {
				t.Error("missing path element")
			}
		})
	}
}

func TestGlyphMultipleGlyphs(t *testing.T) {
	f := parseKern(t)

	// Render several different glyphs at default settings
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("glyph-%d", i), func(t *testing.T) {
			svg, err := svgexp.Glyph(f, i, nil)
			if err != nil {
				t.Logf("glyph %d: %v (may be empty outline)", i, err)
				return
			}

			if !strings.Contains(svg, `<path`) {
				t.Error("missing path element")
			}
			if !strings.Contains(svg, `viewBox="`) {
				t.Error("missing viewBox")
			}
		})
	}
}

func TestGlyphPathCommands(t *testing.T) {
	f := parseTTF(t)

	// 'A' (glyph index 39 in Microsoft-Yahei)
	gid := f.RuneToGlyphID('A')
	if gid == 0 {
		t.Fatal("'A' not mapped")
	}

	svg, err := svgexp.Glyph(f, int(gid), nil)
	if err != nil {
		t.Fatal(err)
	}

	// The path data should contain standard SVG commands
	dStart := strings.Index(svg, `d="`)
	dEnd := strings.Index(svg[dStart+3:], `"`)
	d := svg[dStart+3 : dStart+3+dEnd]

	// TrueType uses quadratic beziers: M, L, Q commands
	hasM := strings.Contains(d, "M")
	hasQ := strings.Contains(d, "Q") || strings.Contains(d, "L")
	if !hasM {
		t.Error("path data missing M (moveto)")
	}
	if !hasQ {
		t.Logf("path data (may have only M+Z for simple glyphs): %q", d)
	}
}

func TestGlyphValidXML(t *testing.T) {
	f := parseTTF(t)

	svg, err := svgexp.Glyph(f, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Basic well-formedness checks
	if !strings.HasPrefix(svg, "<svg") {
		t.Error("SVG should start with <svg")
	}
	if !strings.Contains(svg, "</svg>") {
		t.Error("SVG should have closing </svg>")
	}
	if strings.Count(svg, "<path") != 1 {
		t.Errorf("expected exactly 1 <path>, got %d", strings.Count(svg, "<path"))
	}
}

// --- Helpers ---

func formatScale(s float64) string {
	return fmt.Sprintf("scale-%g", s)
}
