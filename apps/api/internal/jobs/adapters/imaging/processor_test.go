package imaging

import (
	"bytes"
	"context"
	"image"
	"image/color"
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
