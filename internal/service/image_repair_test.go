package service

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func imageUpload(t *testing.T, data []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err = req.ParseMultipartForm(6 << 20); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = req.MultipartForm.RemoveAll() })
	_, header, err := req.FormFile("file")
	if err != nil {
		t.Fatal(err)
	}
	return header
}
func TestImageProcessingPreservesFiveMiBInputsAndBoundsStorage(t *testing.T) {
	var pngData bytes.Buffer
	_ = png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 16, 16)))
	padded := append(pngData.Bytes(), make([]byte, 2<<20)...)
	result, kind, err := ReadAndValidateImage(imageUpload(t, padded), 5<<20)
	if err != nil {
		t.Fatal("previously-supported input rejected", err)
	}
	if len(result) > 256<<10 {
		t.Errorf("stored image=%d bytes exceeds256KiB", len(result))
	}
	if kind != "image/png" {
		t.Errorf("type=%s", kind)
	}
}
func TestImagesRejectCorruptionAndDecodedPixelBombs(t *testing.T) {
	corrupt := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 24)...)
	if _, _, err := ReadAndValidateImage(imageUpload(t, corrupt), 5<<20); err == nil {
		t.Error("corrupt image accepted")
	}
	var large bytes.Buffer
	_ = png.Encode(&large, image.NewGray(image.Rect(0, 0, 3000, 3000)))
	if _, _, err := ReadAndValidateImage(imageUpload(t, large.Bytes()), 5<<20); err == nil {
		t.Error("9MP decoded image accepted")
	}
}
func TestAnimatedGIFRemainsAnimated(t *testing.T) {
	palette := color.Palette{color.Black, color.White}
	a := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	b := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	b.SetColorIndex(1, 1, 1)
	var data bytes.Buffer
	_ = gif.EncodeAll(&data, &gif.GIF{Image: []*image.Paletted{a, b}, Delay: []int{10, 10}, LoopCount: 0})
	result, kind, err := ReadAndValidateImage(imageUpload(t, data.Bytes()), 5<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := gif.DecodeAll(bytes.NewReader(result))
	if err != nil || kind != "image/gif" || len(decoded.Image) != 2 {
		t.Fatal("animation lost", err)
	}
}

func TestAvatarStoredLimitAndDecoderConcurrency(t *testing.T) {
	noisy := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	var state uint32 = 8193
	for i := 0; i < len(noisy.Pix); i += 4 {
		for j := 0; j < 3; j++ {
			state ^= state << 13
			state ^= state >> 17
			state ^= state << 5
			noisy.Pix[i+j] = byte(state)
		}
		noisy.Pix[i+3] = 255
	}
	var data bytes.Buffer
	if err := png.Encode(&data, noisy); err != nil {
		t.Fatal(err)
	}
	output, _, err := ReadAndValidateAvatar(imageUpload(t, data.Bytes()), 5<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) > 128<<10 {
		t.Errorf("avatar stored %d bytes", len(output))
	}
	imageSlots <- struct{}{}
	imageSlots <- struct{}{}
	defer func() { <-imageSlots; <-imageSlots }()
	if _, _, err = ReadAndValidateImage(imageUpload(t, data.Bytes()), 5<<20); err == nil {
		t.Error("third decoder entered while two slots occupied")
	}
}
