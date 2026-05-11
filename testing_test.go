package gofont

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	fontData     []byte
	fontDataOnce sync.Once

	kernFontData     []byte
	kernFontDataOnce sync.Once
)

// loadFont reads the test font file once and caches the result.
func loadFont(t *testing.T) []byte {
	t.Helper()
	fontDataOnce.Do(func() {
		paths := []string{
			"testdata/Microsoft-Yahei.ttf",
			filepath.Join("..", "testdata", "Microsoft-Yahei.ttf"),
		}
		var err error
		for _, p := range paths {
			fontData, err = os.ReadFile(p)
			if err == nil {
				return
			}
		}
		if fontData == nil {
			panic("testdata/Microsoft-Yahei.ttf not found: " + err.Error())
		}
	})
	return fontData
}

// loadKernFont reads a font file with kern/GPOS/GSUB tables.
func loadKernFont(t *testing.T) []byte {
	t.Helper()
	kernFontDataOnce.Do(func() {
		paths := []string{
			"testdata/LEELAWDB.TTF",
			filepath.Join("..", "testdata", "LEELAWDB.TTF"),
		}
		var err error
		for _, p := range paths {
			kernFontData, err = os.ReadFile(p)
			if err == nil {
				return
			}
		}
		if kernFontData == nil {
			panic("testdata/LEELAWDB.TTF not found: " + err.Error())
		}
	})
	return kernFontData
}
