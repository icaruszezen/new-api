package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/webp"
)

// 4x3 lossless WebP (VP8L): opaque blue, red pixel at (0,0), half-transparent green at (3,2).
const ticketTestWebPBase64 = "UklGRjQAAABXRUJQVlA4TCgAAAAvA4AAEBcgEEhSn3wNAUGR/6PNv4AkmS4tL+xTgkA28T1Zizii/9EA"

func newTicketTestImage(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: uint8(x + y), A: 255})
		}
	}
	return img
}

func encodeTicketTestPNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func encodeTicketTestJPEG(t *testing.T, img image.Image, quality int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}))
	return buf.Bytes()
}

func encodeTicketTestGIF(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, gif.Encode(&buf, img, nil))
	return buf.Bytes()
}

func ticketTestPNGChunk(chunkType string, data []byte) []byte {
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	chunk = append(chunk, chunkType...)
	chunk = append(chunk, data...)
	return binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
}

// PNG signature + IHDR only (valid CRC, no pixel data), so only a header parse can succeed.
func ticketTestPNGHeader(width, height uint32) []byte {
	ihdr := binary.BigEndian.AppendUint32(nil, width)
	ihdr = binary.BigEndian.AppendUint32(ihdr, height)
	ihdr = append(ihdr, 8, 6, 0, 0, 0) // 8-bit RGBA, deflate, adaptive filter, no interlace
	return append([]byte("\x89PNG\r\n\x1a\n"), ticketTestPNGChunk("IHDR", ihdr)...)
}

// VP8X container whose canvas size is independent of the VP8L frame header that follows it.
func ticketTestWebPVP8X(canvasWidth, canvasHeight, frameWidth, frameHeight int) []byte {
	vp8x := []byte{0, 0, 0, 0}
	vp8x = append(vp8x, byte(canvasWidth-1), byte((canvasWidth-1)>>8), byte((canvasWidth-1)>>16))
	vp8x = append(vp8x, byte(canvasHeight-1), byte((canvasHeight-1)>>8), byte((canvasHeight-1)>>16))
	vp8l := binary.LittleEndian.AppendUint32([]byte{0x2f}, uint32(frameWidth-1)|uint32(frameHeight-1)<<14)

	body := []byte("WEBP")
	body = append(body, "VP8X"...)
	body = binary.LittleEndian.AppendUint32(body, uint32(len(vp8x)))
	body = append(body, vp8x...)
	body = append(body, "VP8L"...)
	body = binary.LittleEndian.AppendUint32(body, uint32(len(vp8l)))
	body = append(body, vp8l...)
	body = append(body, 0) // pad odd-sized chunk
	riff := binary.LittleEndian.AppendUint32([]byte("RIFF"), uint32(len(body)))
	return append(riff, body...)
}

func TestSanitizeTicketImageAcceptsAllowedFormats(t *testing.T) {
	webpFixture, err := base64.StdEncoding.DecodeString(ticketTestWebPBase64)
	require.NoError(t, err)
	webpImage, err := webp.Decode(bytes.NewReader(webpFixture))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 4, 3), webpImage.Bounds())

	source := newTicketTestImage(40, 30)
	tests := []struct {
		name         string
		raw          []byte
		wantMimeType string
		wantFormat   string
		wantWidth    int
		wantHeight   int
	}{
		{name: "png", raw: encodeTicketTestPNG(t, source), wantMimeType: "image/png", wantFormat: "png", wantWidth: 40, wantHeight: 30},
		{name: "jpeg", raw: encodeTicketTestJPEG(t, source, 90), wantMimeType: "image/jpeg", wantFormat: "jpeg", wantWidth: 40, wantHeight: 30},
		{name: "gif", raw: encodeTicketTestGIF(t, source), wantMimeType: "image/png", wantFormat: "png", wantWidth: 40, wantHeight: 30},
		{name: "webp", raw: webpFixture, wantMimeType: "image/png", wantFormat: "png", wantWidth: 4, wantHeight: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeTicketImage(context.Background(), tt.raw)
			require.NoError(t, err)

			assert.Equal(t, tt.wantMimeType, result.MimeType)
			assert.Equal(t, tt.wantWidth, result.Width)
			assert.Equal(t, tt.wantHeight, result.Height)
			sum := sha256.Sum256(result.Data)
			assert.Equal(t, hex.EncodeToString(sum[:]), result.Sha256)

			config, format, err := image.DecodeConfig(bytes.NewReader(result.Data))
			require.NoError(t, err)
			assert.Equal(t, tt.wantFormat, format)
			assert.Equal(t, tt.wantWidth, config.Width)
			assert.Equal(t, tt.wantHeight, config.Height)
		})
	}

	t.Run("webp pixels survive re-encoding", func(t *testing.T) {
		result, err := SanitizeTicketImage(context.Background(), webpFixture)
		require.NoError(t, err)
		decoded, err := png.Decode(bytes.NewReader(result.Data))
		require.NoError(t, err)
		assert.Equal(t, color.NRGBAModel.Convert(webpImage.At(0, 0)), color.NRGBAModel.Convert(decoded.At(0, 0)))
		assert.Equal(t, color.NRGBAModel.Convert(webpImage.At(3, 2)), color.NRGBAModel.Convert(decoded.At(3, 2)))
	})
}

func TestSanitizeTicketImageRejectsDisallowedTypes(t *testing.T) {
	pngBytes := encodeTicketTestPNG(t, newTicketTestImage(8, 8))
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "svg", raw: []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		{name: "html", raw: []byte(`<!DOCTYPE html><html><body><script>alert(1)</script></body></html>`)},
		{name: "text", raw: []byte("just some text")},
		{name: "pdf", raw: []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n")},
		{name: "empty", raw: []byte{}},
		{name: "bmp", raw: append([]byte("BM"), make([]byte, 64)...)},
		{name: "html prefixed polyglot", raw: append([]byte("<html><script>alert(1)</script>"), pngBytes...)},
		// 只有头部的 16x16 四通道（CMYK）JPEG：若进入完整解码只会得到 ErrTicketImageInvalid
		{name: "cmyk jpeg", raw: []byte{
			0xff, 0xd8, 0xff, 0xc0, 0x00, 0x14, 0x08, 0x00, 0x10, 0x00, 0x10, 0x04,
			0x01, 0x11, 0x00, 0x02, 0x11, 0x00, 0x03, 0x11, 0x00, 0x04, 0x11, 0x00,
			0xff, 0xda, 0x00, 0x0c,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeTicketImage(context.Background(), tt.raw)
			require.ErrorIs(t, err, ErrTicketImageType)
			assert.Nil(t, result)
		})
	}
}

func TestSanitizeTicketImageUploadSizeCap(t *testing.T) {
	pngBytes := encodeTicketTestPNG(t, newTicketTestImage(16, 16))
	atCap := make([]byte, TicketImageMaxUploadBytes)
	copy(atCap, pngBytes)
	overCap := make([]byte, TicketImageMaxUploadBytes+1)
	copy(overCap, pngBytes)

	result, err := SanitizeTicketImage(context.Background(), atCap)
	require.NoError(t, err)
	assert.Equal(t, 16, result.Width)

	result, err = SanitizeTicketImage(context.Background(), overCap)
	require.ErrorIs(t, err, ErrTicketImageTooLarge)
	assert.Nil(t, result)
}

// 头部声明的尺寸超限时必须在完整解码前拒绝：这些输入都没有像素数据，若进入解码只会得到 ErrTicketImageInvalid。
func TestSanitizeTicketImageRejectsDimensionsBeforeDecoding(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "png width over max side", raw: ticketTestPNGHeader(TicketImageMaxSide+1, 10)},
		{name: "png height over max side", raw: ticketTestPNGHeader(10, TicketImageMaxSide+1)},
		{name: "png pixel count over max", raw: ticketTestPNGHeader(6000, 6000)},
		{name: "gif logical screen over max side", raw: []byte("GIF89a\x20\x4e\x01\x00\x00\x00\x00")},
		{name: "webp 1x1 canvas hiding a huge frame", raw: ticketTestWebPVP8X(1, 1, 16383, 16383)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeTicketImage(context.Background(), tt.raw)
			require.ErrorIs(t, err, ErrTicketImageDimensions)
			assert.Nil(t, result)
		})
	}
}

func TestSanitizeTicketImageStripsEmbeddedPayloads(t *testing.T) {
	source := newTicketTestImage(32, 24)
	pngBytes := encodeTicketTestPNG(t, source)
	jpegBytes := encodeTicketTestJPEG(t, source, 90)

	htmlPayload := []byte("<html><script>alert(document.cookie)</script></html>")
	phpPayload := []byte("<?php system($_GET['c']); ?>")
	// APP1 EXIF segment with a fake GPS marker, inserted right after SOI.
	exifBody := append([]byte("Exif\x00\x00MM\x00\x2a\x00\x00\x00\x08"), "GPS 51.5007N 0.1246W"...)
	exifSegment := binary.BigEndian.AppendUint16([]byte{0xff, 0xe1}, uint16(len(exifBody)+2))
	exifSegment = append(exifSegment, exifBody...)

	const pngIHDREnd = 8 + 25
	pngWithTextChunk := append([]byte{}, pngBytes[:pngIHDREnd]...)
	pngWithTextChunk = append(pngWithTextChunk, ticketTestPNGChunk("tEXt", append([]byte("Comment\x00"), phpPayload...))...)
	pngWithTextChunk = append(pngWithTextChunk, pngBytes[pngIHDREnd:]...)

	jpegWithExif := append([]byte{}, jpegBytes[:2]...)
	jpegWithExif = append(jpegWithExif, exifSegment...)
	jpegWithExif = append(jpegWithExif, jpegBytes[2:]...)

	tests := []struct {
		name         string
		raw          []byte
		payload      []byte
		wantMimeType string
	}{
		{name: "html appended after png IEND", raw: append(append([]byte{}, pngBytes...), htmlPayload...), payload: htmlPayload, wantMimeType: "image/png"},
		{name: "php inside png tEXt chunk", raw: pngWithTextChunk, payload: phpPayload, wantMimeType: "image/png"},
		{name: "jpeg exif app1 segment", raw: jpegWithExif, payload: []byte("Exif"), wantMimeType: "image/jpeg"},
		{name: "html appended after jpeg EOI", raw: append(append([]byte{}, jpegBytes...), htmlPayload...), payload: htmlPayload, wantMimeType: "image/jpeg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.True(t, bytes.Contains(tt.raw, tt.payload))

			result, err := SanitizeTicketImage(context.Background(), tt.raw)
			require.NoError(t, err)

			assert.Equal(t, tt.wantMimeType, result.MimeType)
			assert.Equal(t, 32, result.Width)
			assert.Equal(t, 24, result.Height)
			assert.False(t, bytes.Contains(result.Data, tt.payload))
		})
	}
}

func TestSanitizeTicketImageDownscalesToStoredMaxSide(t *testing.T) {
	tests := []struct {
		name         string
		raw          []byte
		wantMimeType string
		wantWidth    int
		wantHeight   int
	}{
		{name: "wide png", raw: encodeTicketTestPNG(t, newTicketTestImage(4000, 1000)), wantMimeType: "image/png", wantWidth: TicketImageStoredMaxSide, wantHeight: 768},
		{name: "tall jpeg", raw: encodeTicketTestJPEG(t, newTicketTestImage(1000, 4000), 85), wantMimeType: "image/jpeg", wantWidth: 768, wantHeight: TicketImageStoredMaxSide},
		{name: "exactly at stored max side is kept", raw: encodeTicketTestPNG(t, newTicketTestImage(TicketImageStoredMaxSide, 16)), wantMimeType: "image/png", wantWidth: TicketImageStoredMaxSide, wantHeight: 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeTicketImage(context.Background(), tt.raw)
			require.NoError(t, err)

			assert.Equal(t, tt.wantMimeType, result.MimeType)
			assert.Equal(t, tt.wantWidth, result.Width)
			assert.Equal(t, tt.wantHeight, result.Height)
			config, _, err := image.DecodeConfig(bytes.NewReader(result.Data))
			require.NoError(t, err)
			assert.Equal(t, tt.wantWidth, config.Width)
			assert.Equal(t, tt.wantHeight, config.Height)
		})
	}
}

// 重新编码的 PNG 超过存储上限时降级为白底 JPEG，透明区域不能变黑。
func TestSanitizeTicketImageFallsBackToJPEGOnWhite(t *testing.T) {
	// 带种子的噪声只是不可压缩的确定性素材，使 PNG 输出必然超过 TicketImageMaxStoredBytes。
	rng := rand.New(rand.NewPCG(1, 2))
	source := image.NewNRGBA(image.Rect(0, 0, 1024, 1024))
	for y := 0; y < 1024; y++ {
		for x := 0; x < 1024; x++ {
			alpha := uint8(255)
			if y < 64 {
				alpha = 0
			}
			source.SetNRGBA(x, y, color.NRGBA{R: uint8(rng.Uint32()), G: uint8(rng.Uint32()), B: uint8(rng.Uint32()), A: alpha})
		}
	}
	raw := encodeTicketTestPNG(t, source)
	require.Greater(t, len(raw), TicketImageMaxStoredBytes)
	require.LessOrEqual(t, len(raw), TicketImageMaxUploadBytes)

	result, err := SanitizeTicketImage(context.Background(), raw)
	require.NoError(t, err)

	assert.Equal(t, "image/jpeg", result.MimeType)
	assert.LessOrEqual(t, len(result.Data), TicketImageMaxStoredBytes)
	decoded, err := jpeg.Decode(bytes.NewReader(result.Data))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 1024, 1024), decoded.Bounds())
	r, g, b, _ := decoded.At(512, 20).RGBA()
	assert.Greater(t, r>>8, uint32(240))
	assert.Greater(t, g>>8, uint32(240))
	assert.Greater(t, b>>8, uint32(240))
}

func TestSanitizeTicketImageRejectsOutputStillTooLargeAfterJPEGFallback(t *testing.T) {
	// 低质量 JPEG 噪声：输入不超过上传上限，但 q65 重新编码后仍超过存储上限。
	rng := rand.New(rand.NewPCG(1, 2))
	source := image.NewRGBA(image.Rect(0, 0, TicketImageStoredMaxSide, TicketImageStoredMaxSide))
	for i := range source.Pix {
		source.Pix[i] = uint8(rng.Uint32())
	}
	for i := 3; i < len(source.Pix); i += 4 {
		source.Pix[i] = 255
	}
	raw := encodeTicketTestJPEG(t, source, 40)
	require.LessOrEqual(t, len(raw), TicketImageMaxUploadBytes)

	result, err := SanitizeTicketImage(context.Background(), raw)
	require.ErrorIs(t, err, ErrTicketImageTooComplex)
	assert.Nil(t, result)
}

func TestSanitizeTicketImageRejectsCorruptImages(t *testing.T) {
	source := newTicketTestImage(64, 48)
	pngBytes := encodeTicketTestPNG(t, source)
	jpegBytes := encodeTicketTestJPEG(t, source, 90)
	gifBytes := encodeTicketTestGIF(t, source)

	corruptedPNG := append([]byte{}, pngBytes...)
	idat := bytes.Index(corruptedPNG, []byte("IDAT"))
	require.Positive(t, idat)
	corruptedPNG[idat+8] ^= 0xff

	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "truncated png", raw: pngBytes[:len(pngBytes)/2]},
		{name: "png with corrupted pixel data", raw: corruptedPNG},
		{name: "truncated jpeg", raw: jpegBytes[:len(jpegBytes)/2]},
		{name: "truncated gif", raw: gifBytes[:len(gifBytes)/2]},
		{name: "png signature only", raw: []byte("\x89PNG\r\n\x1a\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeTicketImage(context.Background(), tt.raw)
			require.ErrorIs(t, err, ErrTicketImageInvalid)
			assert.Nil(t, result)
		})
	}
}

// 扫描段过多的 JPEG 会让标准库解码耗时线性增长；超过上限时必须在解码前拒绝，而少量扫描段照常处理。
func TestSanitizeTicketImageRejectsJPEGWithTooManyScans(t *testing.T) {
	jpegBytes := encodeTicketTestJPEG(t, newTicketTestImage(16, 16), 90)
	sos := bytes.Index(jpegBytes, []byte{0xff, 0xda})
	require.Positive(t, sos)
	scan := jpegBytes[sos : len(jpegBytes)-2] // SOS 段及其熵编码数据，不含 EOI
	withScans := func(count int) []byte {
		raw := append([]byte{}, jpegBytes[:sos]...)
		for i := 0; i < count; i++ {
			raw = append(raw, scan...)
		}
		return append(raw, 0xff, 0xd9)
	}

	accepted, err := SanitizeTicketImage(context.Background(), withScans(ticketJPEGMaxScans))
	require.NoError(t, err)
	assert.Equal(t, "image/jpeg", accepted.MimeType)

	rejected, err := SanitizeTicketImage(context.Background(), withScans(ticketJPEGMaxScans+1))
	require.ErrorIs(t, err, ErrTicketImageInvalid)
	assert.Nil(t, rejected)
}

// APP1 Exif segment whose IFD0 holds a single Orientation (0x0112, SHORT) entry.
func ticketTestExifOrientationSegment(order binary.AppendByteOrder, orientation uint16) []byte {
	tiff := []byte("MM\x00\x2a")
	if order == binary.LittleEndian {
		tiff = []byte("II\x2a\x00")
	}
	tiff = order.AppendUint32(tiff, 8) // IFD0 offset
	tiff = order.AppendUint16(tiff, 1) // entry count
	tiff = order.AppendUint16(tiff, 0x0112)
	tiff = order.AppendUint16(tiff, 3) // SHORT
	tiff = order.AppendUint32(tiff, 1)
	tiff = order.AppendUint16(tiff, orientation)
	tiff = append(tiff, 0, 0)          // value padding
	tiff = order.AppendUint32(tiff, 0) // next IFD
	body := append([]byte("Exif\x00\x00"), tiff...)
	segment := binary.BigEndian.AppendUint16([]byte{0xff, 0xe1}, uint16(len(body)+2))
	return append(segment, body...)
}

// 手机照片常以横向像素 + EXIF Orientation 保存；重新编码会丢弃 EXIF，所以方向必须先应用到像素上。
func TestSanitizeTicketImageAppliesExifOrientation(t *testing.T) {
	// 40x20：左半红、右半蓝
	source := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			pixel := color.NRGBA{R: 255, A: 255}
			if x >= 20 {
				pixel = color.NRGBA{B: 255, A: 255}
			}
			source.SetNRGBA(x, y, pixel)
		}
	}
	jpegBytes := encodeTicketTestJPEG(t, source, 95)
	withExif := func(segment []byte) []byte {
		raw := append([]byte{}, jpegBytes[:2]...)
		raw = append(raw, segment...)
		return append(raw, jpegBytes[2:]...)
	}

	tests := []struct {
		name       string
		raw        []byte
		wantWidth  int
		wantHeight int
		redAt      image.Point
		blueAt     image.Point
	}{
		{name: "no exif", raw: jpegBytes, wantWidth: 40, wantHeight: 20, redAt: image.Pt(10, 10), blueAt: image.Pt(30, 10)},
		{name: "orientation 1", raw: withExif(ticketTestExifOrientationSegment(binary.BigEndian, 1)), wantWidth: 40, wantHeight: 20, redAt: image.Pt(10, 10), blueAt: image.Pt(30, 10)},
		{name: "orientation 6 big endian", raw: withExif(ticketTestExifOrientationSegment(binary.BigEndian, 6)), wantWidth: 20, wantHeight: 40, redAt: image.Pt(10, 10), blueAt: image.Pt(10, 30)},
		{name: "orientation 6 little endian", raw: withExif(ticketTestExifOrientationSegment(binary.LittleEndian, 6)), wantWidth: 20, wantHeight: 40, redAt: image.Pt(10, 10), blueAt: image.Pt(10, 30)},
		{name: "orientation 8", raw: withExif(ticketTestExifOrientationSegment(binary.BigEndian, 8)), wantWidth: 20, wantHeight: 40, redAt: image.Pt(10, 30), blueAt: image.Pt(10, 10)},
		{name: "orientation out of range is ignored", raw: withExif(ticketTestExifOrientationSegment(binary.BigEndian, 9)), wantWidth: 40, wantHeight: 20, redAt: image.Pt(10, 10), blueAt: image.Pt(30, 10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeTicketImage(context.Background(), tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.wantWidth, result.Width)
			assert.Equal(t, tt.wantHeight, result.Height)
			assert.False(t, bytes.Contains(result.Data, []byte("Exif")))

			decoded, err := jpeg.Decode(bytes.NewReader(result.Data))
			require.NoError(t, err)
			require.Equal(t, image.Rect(0, 0, tt.wantWidth, tt.wantHeight), decoded.Bounds())
			r, _, b, _ := decoded.At(tt.redAt.X, tt.redAt.Y).RGBA()
			assert.Greater(t, r>>8, uint32(200), "red expected at %v", tt.redAt)
			assert.Less(t, b>>8, uint32(60), "red expected at %v", tt.redAt)
			r, _, b, _ = decoded.At(tt.blueAt.X, tt.blueAt.Y).RGBA()
			assert.Less(t, r>>8, uint32(60), "blue expected at %v", tt.blueAt)
			assert.Greater(t, b>>8, uint32(200), "blue expected at %v", tt.blueAt)
		})
	}
}

// EXIF 来自用户上传，任何截断或格式错误都只能退回“不旋转”，不能越界。
func TestJPEGExifOrientationRejectsMalformedSegments(t *testing.T) {
	valid := ticketTestExifOrientationSegment(binary.BigEndian, 6)
	jpegWith := func(segment []byte) []byte {
		return append(append([]byte{0xff, 0xd8}, segment...), 0xff, 0xda, 0x00, 0x02)
	}
	wrongType := append([]byte{}, valid...)
	wrongType[4+6+8+2+3] = 4 // LONG instead of SHORT
	hugeIFDOffset := append([]byte{}, valid...)
	copy(hugeIFDOffset[4+6+4:], []byte{0xff, 0xff, 0xff, 0xf0})
	segmentPastEnd := append([]byte{}, valid...)
	segmentPastEnd[3] += 10
	truncatedIFD := append([]byte{}, valid[:len(valid)-10]...)
	binary.BigEndian.PutUint16(truncatedIFD[2:], uint16(len(truncatedIFD)-2))

	tests := []struct {
		name string
		raw  []byte
		want int
	}{
		{name: "valid", raw: jpegWith(valid), want: 6},
		{name: "not a jpeg", raw: valid, want: 1},
		{name: "tag type is not SHORT", raw: jpegWith(wrongType), want: 1},
		{name: "IFD offset past the segment", raw: jpegWith(hugeIFDOffset), want: 1},
		{name: "segment length past the end", raw: jpegWith(segmentPastEnd), want: 1},
		{name: "truncated IFD entry", raw: jpegWith(truncatedIFD), want: 1},
		{name: "exif after start of scan is ignored", raw: append([]byte{0xff, 0xd8, 0xff, 0xda, 0x00, 0x02}, valid...), want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, jpegExifOrientation(tt.raw))
		})
	}
}

// 等待解码槽位时请求被取消（客户端断开）要立即返回，而不是继续排队。
func TestSanitizeTicketImageStopsWaitingWhenContextCancelled(t *testing.T) {
	for range ticketImageDecodeConcurrency {
		ticketImageSlots <- struct{}{}
	}
	t.Cleanup(func() {
		for range ticketImageDecodeConcurrency {
			<-ticketImageSlots
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := SanitizeTicketImage(ctx, encodeTicketTestPNG(t, newTicketTestImage(8, 8)))
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, result)
}
