// Package imaging implements image transformations using Go's standard library.
package imaging

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"

	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

// Processor decodes, resizes, and encodes JPEG and PNG images.
type Processor struct{}

const maxPixels = 40_000_000

// NewProcessor creates a standard-library image processor.
func NewProcessor() *Processor { return &Processor{} }

// Process applies operations in request order.
func (*Processor) Process(ctx context.Context, source []byte, operations []domain.Operation) ([]byte, domain.Output, error) {
	configuration, _, err := image.DecodeConfig(bytes.NewReader(source))
	if err != nil {
		return nil, domain.Output{}, fmt.Errorf("inspect source image: %w", err)
	}
	if configuration.Width <= 0 || configuration.Height <= 0 || int64(configuration.Width)*int64(configuration.Height) > maxPixels {
		return nil, domain.Output{}, fmt.Errorf("source image exceeds %d decoded pixels", maxPixels)
	}
	current, sourceFormat, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, domain.Output{}, fmt.Errorf("decode source image: %w", err)
	}
	format := normalizeFormat(sourceFormat)
	quality := 85
	for _, operation := range operations {
		if err := ctx.Err(); err != nil {
			return nil, domain.Output{}, err
		}
		switch operation.Type {
		case "resize":
			width, height := dimensions(current.Bounds().Dx(), current.Bounds().Dy(), operation.Width, operation.Height)
			if int64(width)*int64(height) > maxPixels {
				return nil, domain.Output{}, fmt.Errorf("output image exceeds %d pixels", maxPixels)
			}
			current = resize(current, width, height)
		case "convert":
			format = operation.Format
			if operation.Quality > 0 {
				quality = operation.Quality
			}
		}
	}
	if format != "jpeg" && format != "png" {
		return nil, domain.Output{}, fmt.Errorf("unsupported output format %q", format)
	}
	var buffer bytes.Buffer
	contentType := "image/png"
	if format == "jpeg" {
		contentType = "image/jpeg"
		if err := jpeg.Encode(&buffer, current, &jpeg.Options{Quality: quality}); err != nil {
			return nil, domain.Output{}, fmt.Errorf("encode jpeg: %w", err)
		}
	} else if err := png.Encode(&buffer, current); err != nil {
		return nil, domain.Output{}, fmt.Errorf("encode png: %w", err)
	}
	bounds := current.Bounds()
	return buffer.Bytes(), domain.Output{ContentType: contentType, Format: format, SizeBytes: int64(buffer.Len()), Width: bounds.Dx(), Height: bounds.Dy()}, nil
}

func normalizeFormat(format string) string {
	if format == "jpg" {
		return "jpeg"
	}
	return format
}

func dimensions(sourceWidth, sourceHeight, width, height int) (int, int) {
	if width <= 0 {
		width = max(1, sourceWidth*height/sourceHeight)
	}
	if height <= 0 {
		height = max(1, sourceHeight*width/sourceWidth)
	}
	return width, height
}

func resize(source image.Image, width, height int) image.Image {
	destination := image.NewRGBA(image.Rect(0, 0, width, height))
	bounds := source.Bounds()
	for y := range height {
		sy := bounds.Min.Y + y*bounds.Dy()/height
		for x := range width {
			sx := bounds.Min.X + x*bounds.Dx()/width
			destination.Set(x, y, source.At(sx, sy))
		}
	}
	draw.Draw(destination, destination.Bounds(), destination, image.Point{}, draw.Src)
	return destination
}
