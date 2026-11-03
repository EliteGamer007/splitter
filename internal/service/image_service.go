package service

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"mime/multipart"
)

const MaxImageInput int64 = 5 << 20
const MaxImagePixels int64 = 8_000_000
const MaxStoredImage = 256 << 10
const MaxStoredAvatar = 128 << 10

var imageSlots = make(chan struct{}, 2)

// ReadAndValidateImage accepts the established 5MiB input limit and stores a bounded derivative.
func ReadAndValidateImage(file *multipart.FileHeader, maxSize int64) ([]byte, string, error) {
	return readImage(file, maxSize, MaxStoredImage)
}
func ReadAndValidateAvatar(file *multipart.FileHeader, maxSize int64) ([]byte, string, error) {
	return readImage(file, maxSize, MaxStoredAvatar)
}
func readImage(file *multipart.FileHeader, maxSize int64, storedLimit int) ([]byte, string, error) {
	if file == nil {
		return nil, "", fmt.Errorf("missing image")
	}
	if maxSize <= 0 || maxSize > MaxImageInput {
		maxSize = MaxImageInput
	}
	if file.Size > maxSize {
		return nil, "", fmt.Errorf("image exceeds %d MiB input limit", maxSize>>20)
	}
	// Do not allow queued decoders to retain an unbounded number of large uploads.
	select {
	case imageSlots <- struct{}{}:
		defer func() { <-imageSlots }()
	default:
		return nil, "", fmt.Errorf("image processing busy; retry shortly")
	}
	source, err := file.Open()
	if err != nil {
		return nil, "", fmt.Errorf("unable to read image")
	}
	defer source.Close()
	data, err := io.ReadAll(io.LimitReader(source, maxSize+1))
	if err != nil {
		return nil, "", fmt.Errorf("unable to read image")
	}
	if int64(len(data)) > maxSize {
		return nil, "", fmt.Errorf("image exceeds input limit")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("invalid or corrupt image")
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width) > MaxImagePixels/int64(config.Height) {
		return nil, "", fmt.Errorf("image exceeds 8 megapixel decoded limit")
	}
	if format != "png" && format != "jpeg" && format != "gif" {
		return nil, "", fmt.Errorf("only JPEG, PNG and GIF photos are supported")
	}
	if format == "gif" {
		if err = validateGIFFrames(data); err != nil {
			return nil, "", err
		}
		animation, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			return nil, "", fmt.Errorf("invalid or corrupt GIF")
		}
		if len(data) <= storedLimit {
			return data, "image/gif", nil
		}
		var compressed bytes.Buffer
		if err = gif.EncodeAll(&compressed, animation); err != nil {
			return nil, "", fmt.Errorf("unable to process GIF")
		}
		if compressed.Len() > storedLimit {
			return nil, "", fmt.Errorf("animated GIF exceeds %d KiB stored image limit; use a smaller animation", storedLimit>>10)
		}
		return compressed.Bytes(), "image/gif", nil
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("invalid or corrupt image")
	}
	mediaType := "image/" + format
	if len(data) <= storedLimit {
		return data, mediaType, nil
	}
	// Preserve PNG alpha; JPEG input remains JPEG. Reduce dimensions gradually if needed.
	width, height := config.Width, config.Height
	if width > 1600 || height > 1600 {
		ratio := math.Min(1600/float64(width), 1600/float64(height))
		width = int(float64(width) * ratio)
		height = int(float64(height) * ratio)
	}
	for attempt := 0; attempt < 16; attempt++ {
		resized := resizeImage(decoded, width, height)
		var output bytes.Buffer
		if format == "png" {
			err = png.Encode(&output, resized)
		} else {
			err = jpeg.Encode(&output, resized, &jpeg.Options{Quality: 82})
		}
		if err != nil {
			return nil, "", fmt.Errorf("unable to process image")
		}
		if output.Len() <= storedLimit {
			return output.Bytes(), mediaType, nil
		}
		width = int(float64(width) * 0.78)
		height = int(float64(height) * 0.78)
		if width < 1 {
			width = 1
		}
		if height < 1 {
			height = 1
		}
	}
	return nil, "", fmt.Errorf("image cannot fit the stored image budget")
}
func resizeImage(source image.Image, width, height int) *image.NRGBA {
	output := image.NewNRGBA(image.Rect(0, 0, width, height))
	bounds := source.Bounds()
	// Bilinear interpolation in premultiplied space avoids dark edges around transparency.
	for y := 0; y < height; y++ {
		sy := (float64(y)+0.5)*float64(bounds.Dy())/float64(height) - 0.5
		if sy < 0 {
			sy = 0
		}
		y0 := int(sy)
		y1 := y0 + 1
		if y1 >= bounds.Dy() {
			y1 = bounds.Dy() - 1
		}
		fy := sy - float64(y0)
		for x := 0; x < width; x++ {
			sx := (float64(x)+0.5)*float64(bounds.Dx())/float64(width) - 0.5
			if sx < 0 {
				sx = 0
			}
			x0 := int(sx)
			x1 := x0 + 1
			if x1 >= bounds.Dx() {
				x1 = bounds.Dx() - 1
			}
			fx := sx - float64(x0)
			points := [][2]int{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}}
			weights := []float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy}
			var r, g, b, a float64
			for i, p := range points {
				pr, pg, pb, pa := source.At(bounds.Min.X+p[0], bounds.Min.Y+p[1]).RGBA()
				r += float64(pr) * weights[i]
				g += float64(pg) * weights[i]
				b += float64(pb) * weights[i]
				a += float64(pa) * weights[i]
			}
			pixel := color.NRGBA{A: uint8(a / 257)}
			if a > 0 {
				pixel.R = uint8(math.Min(255, r*255/a))
				pixel.G = uint8(math.Min(255, g*255/a))
				pixel.B = uint8(math.Min(255, b*255/a))
			}
			output.SetNRGBA(x, y, pixel)
		}
	}
	return output
}

// Count frame allocations before gif.DecodeAll can allocate them. A tiny compressed
// animation must not expand to an unbounded number of full image frames.
func validateGIFFrames(data []byte) error {
	invalid := fmt.Errorf("invalid or corrupt GIF")
	if len(data) < 13 {
		return invalid
	}
	offset := 13
	if data[10]&0x80 != 0 {
		offset += 3 * (1 << ((data[10] & 7) + 1))
	}
	skipBlocks := func() bool {
		for {
			if offset >= len(data) {
				return false
			}
			size := int(data[offset])
			offset++
			if size == 0 {
				return true
			}
			offset += size
			if offset > len(data) {
				return false
			}
		}
	}
	var pixels int64
	for offset < len(data) {
		marker := data[offset]
		offset++
		switch marker {
		case 0x3b:
			return nil
		case 0x21:
			offset++
			if !skipBlocks() {
				return invalid
			}
		case 0x2c:
			if offset+9 > len(data) {
				return invalid
			}
			width := int64(binary.LittleEndian.Uint16(data[offset+4:]))
			height := int64(binary.LittleEndian.Uint16(data[offset+6:]))
			pixels += width * height
			if width < 1 || height < 1 || pixels > MaxImagePixels {
				return fmt.Errorf("GIF exceeds 8 megapixel decoded frame limit")
			}
			packed := data[offset+8]
			offset += 9
			if packed&0x80 != 0 {
				offset += 3 * (1 << ((packed & 7) + 1))
			}
			offset++
			if !skipBlocks() {
				return invalid
			}
		default:
			return invalid
		}
	}
	return invalid
}
