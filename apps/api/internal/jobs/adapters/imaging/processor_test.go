package imaging

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

func TestProcessResizesAndConvertsImage(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 40, 20))
	source.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}

	data, output, err := NewProcessor().Process(context.Background(), encoded.Bytes(), []domain.Operation{
		{Type: "resize", Width: 10}, {Type: "convert", Format: "jpeg", Quality: 80},
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if output.Width != 10 || output.Height != 5 || output.Format != "jpeg" || output.ContentType != "image/jpeg" {
		t.Fatalf("output = %#v", output)
	}
	decoded, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if format != "jpeg" || decoded.Bounds().Dx() != 10 || decoded.Bounds().Dy() != 5 {
		t.Fatalf("result format=%s bounds=%v", format, decoded.Bounds())
	}
}

func TestProcessContainCoverAndCrop(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 40, 20))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name          string
		operations    []domain.Operation
		width, height int
	}{
		{name: "contain", operations: []domain.Operation{{Type: "resize", Width: 10, Height: 10, Fit: "contain"}}, width: 10, height: 5},
		{name: "cover", operations: []domain.Operation{{Type: "resize", Width: 10, Height: 10, Fit: "cover"}}, width: 10, height: 10},
		{name: "crop", operations: []domain.Operation{{Type: "crop", X: 5, Y: 2, Width: 12, Height: 8}}, width: 12, height: 8},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, output, err := NewProcessor().Process(context.Background(), encoded.Bytes(), test.operations)
			if err != nil {
				t.Fatal(err)
			}
			if output.Width != test.width || output.Height != test.height {
				t.Fatalf("output = %dx%d", output.Width, output.Height)
			}
		})
	}
}

func TestProcessAppliesJPEGEXIFOrientationAndStripsMetadata(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 3, 2))
	source.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, source, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	exif := []byte{0xff, 0xe1, 0x00, 0x22, 'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	withEXIF := append(append([]byte{}, data[:2]...), append(exif, data[2:]...)...)
	result, output, err := NewProcessor().Process(context.Background(), withEXIF, []domain.Operation{{Type: "convert", Format: "jpeg", Quality: 90}})
	if err != nil {
		t.Fatal(err)
	}
	if output.Width != 2 || output.Height != 3 {
		t.Fatalf("oriented output = %dx%d", output.Width, output.Height)
	}
	if bytes.Contains(result, []byte("Exif\x00\x00")) {
		t.Fatal("encoded result retained EXIF metadata")
	}
}
