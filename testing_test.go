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
)

// loadFont reads the test font file once and caches the result.
func loadFont(t *testing.T) []byte {
	t.Helper()
	fontDataOnce.Do(func() {
		paths := []string{
			"fonts/fonteditor.ttf",
			filepath.Join("..", "fonts", "fonteditor.ttf"),
		}
		var err error
		for _, p := range paths {
			fontData, err = os.ReadFile(p)
			if err == nil {
				return
			}
		}
		if fontData == nil {
			panic("fonts/fonteditor.ttf not found: " + err.Error())
		}
	})
	return fontData
}
