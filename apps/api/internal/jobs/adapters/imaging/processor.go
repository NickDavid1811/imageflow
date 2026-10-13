// Package imaging implements image transformations using Go's standard library.
package imaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"

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
	if normalizeFormat(sourceFormat) == "jpeg" {
		current = orient(current, jpegOrientation(source))
	}
	format := normalizeFormat(sourceFormat)
	quality := 85
	for _, operation := range operations {
		if err := ctx.Err(); err != nil {
			return nil, domain.Output{}, err
		}
		switch operation.Type {
		case "resize":
			width, height := dimensions(current.Bounds().Dx(), current.Bounds().Dy(), operation.Width, operation.Height, operation.Fit)
			if int64(width)*int64(height) > maxPixels {
				return nil, domain.Output{}, fmt.Errorf("output image exceeds %d pixels", maxPixels)
			}
			if operation.Fit == "cover" {
				scale := math.Max(float64(operation.Width)/float64(current.Bounds().Dx()), float64(operation.Height)/float64(current.Bounds().Dy()))
				scaledWidth, scaledHeight := max(1, int(math.Round(float64(current.Bounds().Dx())*scale))), max(1, int(math.Round(float64(current.Bounds().Dy())*scale)))
				if int64(scaledWidth)*int64(scaledHeight) > maxPixels {
					return nil, domain.Output{}, fmt.Errorf("intermediate image exceeds %d pixels", maxPixels)
				}
				current = resize(current, scaledWidth, scaledHeight)
				current, err = crop(current, (current.Bounds().Dx()-operation.Width)/2, (current.Bounds().Dy()-operation.Height)/2, operation.Width, operation.Height)
				if err != nil {
					return nil, domain.Output{}, err
				}
			} else {
				current = resize(current, width, height)
			}
		case "crop":
			current, err = crop(current, operation.X, operation.Y, operation.Width, operation.Height)
			if err != nil {
				return nil, domain.Output{}, err
			}
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

func dimensions(sourceWidth, sourceHeight, width, height int, fit string) (int, int) {
	if width <= 0 {
		return max(1, int(math.Round(float64(sourceWidth)*float64(height)/float64(sourceHeight)))), height
	}
	if height <= 0 {
		return width, max(1, int(math.Round(float64(sourceHeight)*float64(width)/float64(sourceWidth))))
	}
	if fit == "" || fit == "contain" {
		scale := math.Min(float64(width)/float64(sourceWidth), float64(height)/float64(sourceHeight))
		return max(1, int(math.Round(float64(sourceWidth)*scale))), max(1, int(math.Round(float64(sourceHeight)*scale)))
	}
	return width, height
}

func resize(source image.Image, width, height int) image.Image {
	destination := image.NewRGBA(image.Rect(0, 0, width, height))
	bounds := source.Bounds()
	for y := range height {
		sy := (float64(y)+0.5)*float64(bounds.Dy())/float64(height) - 0.5
		y0 := max(0, min(bounds.Dy()-1, int(math.Floor(sy))))
		y1 := min(bounds.Dy()-1, y0+1)
		fy := math.Max(0, sy-float64(y0))
		for x := range width {
			sx := (float64(x)+0.5)*float64(bounds.Dx())/float64(width) - 0.5
			x0 := max(0, min(bounds.Dx()-1, int(math.Floor(sx))))
			x1 := min(bounds.Dx()-1, x0+1)
			fx := math.Max(0, sx-float64(x0))
			destination.Set(x, y, bilinear(source, bounds.Min.X+x0, bounds.Min.Y+y0, bounds.Min.X+x1, bounds.Min.Y+y1, fx, fy))
		}
	}
	return destination
}

func bilinear(source image.Image, x0, y0, x1, y1 int, fx, fy float64) color.NRGBA64 {
	c00 := color.NRGBA64Model.Convert(source.At(x0, y0)).(color.NRGBA64)
	c10 := color.NRGBA64Model.Convert(source.At(x1, y0)).(color.NRGBA64)
	c01 := color.NRGBA64Model.Convert(source.At(x0, y1)).(color.NRGBA64)
	c11 := color.NRGBA64Model.Convert(source.At(x1, y1)).(color.NRGBA64)
	value := func(v00, v10, v01, v11 uint32) uint16 {
		top := float64(v00)*(1-fx) + float64(v10)*fx
		bottom := float64(v01)*(1-fx) + float64(v11)*fx
		return uint16(math.Round(top*(1-fy) + bottom*fy))
	}
	return color.NRGBA64{R: value(uint32(c00.R), uint32(c10.R), uint32(c01.R), uint32(c11.R)), G: value(uint32(c00.G), uint32(c10.G), uint32(c01.G), uint32(c11.G)), B: value(uint32(c00.B), uint32(c10.B), uint32(c01.B), uint32(c11.B)), A: value(uint32(c00.A), uint32(c10.A), uint32(c01.A), uint32(c11.A))}
}

func crop(source image.Image, x, y, width, height int) (image.Image, error) {
	bounds := source.Bounds()
	if x < 0 || y < 0 || width <= 0 || height <= 0 || x+width > bounds.Dx() || y+height > bounds.Dy() {
		return nil, fmt.Errorf("crop rectangle exceeds image bounds")
	}
	destination := image.NewRGBA(image.Rect(0, 0, width, height))
	for dy := range height {
		for dx := range width {
			destination.Set(dx, dy, source.At(bounds.Min.X+x+dx, bounds.Min.Y+y+dy))
		}
	}
	return destination, nil
}

func orient(source image.Image, orientation int) image.Image {
	if orientation < 2 || orientation > 8 {
		return source
	}
	b := source.Bounds()
	width, height := b.Dx(), b.Dy()
	dw, dh := width, height
	if orientation >= 5 {
		dw, dh = height, width
	}
	destination := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := range dh {
		for x := range dw {
			sx, sy := x, y
			switch orientation {
			case 2:
				sx = width - 1 - x
			case 3:
				sx, sy = width-1-x, height-1-y
			case 4:
				sy = height - 1 - y
			case 5:
				sx, sy = y, x
			case 6:
				sx, sy = y, height-1-x
			case 7:
				sx, sy = width-1-y, height-1-x
			case 8:
				sx, sy = width-1-y, x
			}
			destination.Set(x, y, source.At(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	return destination
}

func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 1
	}
	for offset := 2; offset+4 <= len(data); {
		if data[offset] != 0xff {
			break
		}
		marker := data[offset+1]
		offset += 2
		if marker == 0xda || marker == 0xd9 {
			break
		}
		if offset+2 > len(data) {
			break
		}
		size := int(binary.BigEndian.Uint16(data[offset : offset+2]))
		if size < 2 || offset+size > len(data) {
			break
		}
		segment := data[offset+2 : offset+size]
		if marker == 0xe1 && len(segment) >= 14 && string(segment[:6]) == "Exif\x00\x00" {
			if value := tiffOrientation(segment[6:]); value >= 1 && value <= 8 {
				return value
			}
		}
		offset += size
	}
	return 1
}

func tiffOrientation(data []byte) int {
	if len(data) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(data[2:4]) != 42 {
		return 1
	}
	offset := int(order.Uint32(data[4:8]))
	if offset < 0 || offset+2 > len(data) {
		return 1
	}
	count := int(order.Uint16(data[offset : offset+2]))
	offset += 2
	for index := 0; index < count; index++ {
		entry := offset + index*12
		if entry+12 > len(data) {
			break
		}
		if order.Uint16(data[entry:entry+2]) == 0x0112 && order.Uint16(data[entry+2:entry+4]) == 3 && order.Uint32(data[entry+4:entry+8]) == 1 {
			return int(order.Uint16(data[entry+8 : entry+10]))
		}
	}
	return 1
}
