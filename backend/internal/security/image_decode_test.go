package security_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
)

func TestPocketBaseImagingRejectsInvalidTIFFPaletteReferences(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "invalid-palette-ref.tiff"))
	if err != nil {
		t.Fatal(err)
	}

	img, err := imaging.Decode(bytes.NewReader(data))
	if err == nil {
		t.Fatal("imaging.Decode accepted a TIFF with an invalid palette reference")
	}
	if img != nil {
		t.Fatalf("imaging.Decode returned an image after rejecting the TIFF: %T", img)
	}
}
