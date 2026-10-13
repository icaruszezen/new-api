package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"runtime"
	"time"

	"github.com/QuantumNous/new-api/common"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/riff"
	"golang.org/x/image/vp8"
	"golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"
)

// 工单图片上传限制
const (
	TicketImageMaxUploadBytes = 8 << 20
	TicketImageMaxStoredBytes = 3 << 20
	TicketImageMaxSide        = 12000
	TicketImageMaxPixels      = 25_000_000
	TicketImageStoredMaxSide  = 3072
)

const (
	// 单张大图解码时的工作内存可达数百 MB，同一时刻只处理一张，峰值内存与并发请求数无关
	ticketImageDecodeConcurrency = 1
	// 等待解码槽位的上限，超时直接告知用户稍后重试，避免请求无限排队
	ticketImageSlotWait = 30 * time.Second
	// 正常照片和截图的 JPEG 扫描段不超过二十个左右
	ticketJPEGMaxScans = 64
)

var (
	ErrTicketImageTooLarge   = errors.New("ticket image is too large")
	ErrTicketImageType       = errors.New("ticket image type is not allowed")
	ErrTicketImageDimensions = errors.New("ticket image dimensions are not allowed")
	ErrTicketImageInvalid    = errors.New("ticket image is invalid")
	ErrTicketImageTooComplex = errors.New("ticket image cannot be stored within the size limit")
	ErrTicketImageBusy       = errors.New("ticket image processing is busy")
)

// TicketImage 是重新编码后的工单图片，只包含从像素重新生成的字节
type TicketImage struct {
	Data     []byte
	MimeType string
	Width    int
	Height   int
	Sha256   string
}

// 魔数嗅探结果 -> image.DecodeConfig 返回的格式名
var ticketImageFormats = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpeg",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// 进程级解码/编码并发上限，限制并发上传时的峰值内存
var ticketImageSlots = make(chan struct{}, ticketImageDecodeConcurrency)

// SanitizeTicketImage 校验用户上传的图片并从像素重新编码，丢弃 EXIF/GPS 等元数据、
// 文件尾部追加的内容以及任何多态文件结构。客户端提供的文件名和 Content-Type 一律不参与判断。
// JPEG 的 EXIF 方向会先应用到像素上再丢弃。
func SanitizeTicketImage(ctx context.Context, raw []byte) (result *TicketImage, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			common.SysError(fmt.Sprintf("ticket image decode panic: %v", recovered))
			result, err = nil, ErrTicketImageInvalid
		}
	}()

	if len(raw) > TicketImageMaxUploadBytes {
		return nil, ErrTicketImageTooLarge
	}
	format, ok := ticketImageFormats[http.DetectContentType(raw)]
	if !ok {
		return nil, ErrTicketImageType
	}
	config, decodedFormat, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrTicketImageInvalid
	}
	if decodedFormat != format {
		return nil, ErrTicketImageType
	}
	// CMYK/YCCK JPEG 解码时的内存开销远高于普通 JPEG，且几乎不会出现在截图和手机照片中
	if config.ColorModel == color.CMYKModel {
		return nil, ErrTicketImageType
	}
	// 完整解码前检查尺寸，防止解压炸弹
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > TicketImageMaxSide || config.Height > TicketImageMaxSide ||
		config.Width*config.Height > TicketImageMaxPixels {
		return nil, ErrTicketImageDimensions
	}
	if format == "webp" {
		frameWidth, frameHeight, err := webpFrameSize(raw)
		if err != nil {
			return nil, ErrTicketImageInvalid
		}
		if frameWidth != config.Width || frameHeight != config.Height {
			return nil, ErrTicketImageDimensions
		}
	}

	orientation := 1
	if format == "jpeg" {
		// 标准库解码渐进式 JPEG 不限制扫描段数量，耗时随段数线性增长，几 MB 的构造文件可占用解码槽数十分钟。
		// 熵编码数据中的 0xFF 后面只会跟 0x00 或 RST 标记，FF DA 只可能是 SOS 标记（或元数据里的偶然字节，只会让计数偏大）。
		if bytes.Count(raw, []byte{0xFF, 0xDA}) > ticketJPEGMaxScans {
			return nil, ErrTicketImageInvalid
		}
		orientation = jpegExifOrientation(raw)
	}

	slotTimer := time.NewTimer(ticketImageSlotWait)
	defer slotTimer.Stop()
	select {
	case ticketImageSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-slotTimer.C:
		return nil, ErrTicketImageBusy
	}
	defer func() {
		// 释放槽位前回收本次解码产生的垃圾：否则 GC 按上一张图的峰值放宽堆目标，
		// 下一张图的峰值内存会接近翻倍
		runtime.GC()
		<-ticketImageSlots
	}()

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrTicketImageInvalid
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > TicketImageStoredMaxSide || height > TicketImageStoredMaxSide {
		if width >= height {
			height = max(1, (height*TicketImageStoredMaxSide+width/2)/width)
			width = TicketImageStoredMaxSide
		} else {
			width = max(1, (width*TicketImageStoredMaxSide+height/2)/height)
			height = TicketImageStoredMaxSide
		}
		// CatmullRom 会额外分配 目标宽 x 源高 x 32 字节的临时缓冲（大图可达数百 MB），双线性不需要
		scaled := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), img, bounds, draw.Src, nil)
		img = scaled
	}
	// 在缩小后的图上旋转，避免为原尺寸图片再复制一份
	if orientation != 1 {
		img = applyExifOrientation(img, orientation)
		width, height = img.Bounds().Dx(), img.Bounds().Dy()
	}

	var buf bytes.Buffer
	mimeType := "image/png"
	if format == "jpeg" {
		mimeType = "image/jpeg"
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 88})
	} else {
		err = png.Encode(&buf, img)
	}
	if err != nil {
		return nil, ErrTicketImageInvalid
	}
	if buf.Len() > TicketImageMaxStoredBytes {
		// 转 JPEG 前先铺白底，避免透明区域变黑
		flattened := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(flattened, flattened.Bounds(), image.White, image.Point{}, draw.Src)
		draw.Draw(flattened, flattened.Bounds(), img, img.Bounds().Min, draw.Over)
		mimeType = "image/jpeg"
		for _, quality := range []int{80, 65} {
			buf.Reset()
			if err := jpeg.Encode(&buf, flattened, &jpeg.Options{Quality: quality}); err != nil {
				return nil, ErrTicketImageInvalid
			}
			if buf.Len() <= TicketImageMaxStoredBytes {
				break
			}
		}
		if buf.Len() > TicketImageMaxStoredBytes {
			return nil, ErrTicketImageTooComplex
		}
	}

	data := buf.Bytes()
	sum := sha256.Sum256(data)
	return &TicketImage{
		Data:     data,
		MimeType: mimeType,
		Width:    width,
		Height:   height,
		Sha256:   hex.EncodeToString(sum[:]),
	}, nil
}

// jpegExifOrientation 读取 JPEG 中 EXIF 的 Orientation 标签（2..8），
// 缺失、不合法或任何解析失败都返回 1（不旋转）。
func jpegExifOrientation(raw []byte) int {
	if len(raw) < 4 || raw[0] != 0xff || raw[1] != 0xd8 {
		return 1
	}
	// 逐段查找 APP1 "Exif\x00\x00"，遇到 SOS/EOI 说明元数据段已经结束
	pos := 2
	var tiff []byte
	for tiff == nil {
		if pos+4 > len(raw) || raw[pos] != 0xff {
			return 1
		}
		marker := raw[pos+1]
		if marker == 0xda || marker == 0xd9 {
			return 1
		}
		segmentEnd := pos + 2 + int(binary.BigEndian.Uint16(raw[pos+2:]))
		if segmentEnd < pos+4 || segmentEnd > len(raw) {
			return 1
		}
		segment := raw[pos+4 : segmentEnd]
		if marker == 0xe1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			tiff = segment[6:]
		}
		pos = segmentEnd
	}

	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:]) != 42 {
		return 1
	}
	ifdOffset := uint64(order.Uint32(tiff[4:]))
	if ifdOffset+2 > uint64(len(tiff)) {
		return 1
	}
	ifd := tiff[ifdOffset:]
	entryCount := int(order.Uint16(ifd))
	for i := 0; i < entryCount; i++ {
		entry := ifd[min(2+i*12, len(ifd)):]
		if len(entry) < 12 {
			return 1
		}
		// 0x0112 Orientation，类型必须是 SHORT(3)
		if order.Uint16(entry) != 0x0112 {
			continue
		}
		if order.Uint16(entry[2:]) != 3 {
			return 1
		}
		orientation := int(order.Uint16(entry[8:]))
		if orientation < 2 || orientation > 8 {
			return 1
		}
		return orientation
	}
	return 1
}

// applyExifOrientation 按 EXIF Orientation（2..8）翻转或旋转像素，返回原点在 (0,0) 的新图；
// 5..8 会交换宽高。
func applyExifOrientation(src image.Image, orientation int) image.Image {
	bounds := src.Bounds()
	w, h := float64(bounds.Dx()), float64(bounds.Dy())
	minX, minY := float64(bounds.Min.X), float64(bounds.Min.Y)
	// 源坐标 (x, y) 到目标坐标：dstX = a*x + b*y + c，dstY = d*x + e*y + f
	var transform f64.Aff3
	switch orientation {
	case 2: // 水平翻转
		transform = f64.Aff3{-1, 0, w + minX, 0, 1, -minY}
	case 3: // 旋转 180°
		transform = f64.Aff3{-1, 0, w + minX, 0, -1, h + minY}
	case 4: // 垂直翻转
		transform = f64.Aff3{1, 0, -minX, 0, -1, h + minY}
	case 5: // 沿主对角线翻转
		transform = f64.Aff3{0, 1, -minY, 1, 0, -minX}
	case 6: // 顺时针旋转 90°
		transform = f64.Aff3{0, -1, h + minY, 1, 0, -minX}
	case 7: // 沿副对角线翻转
		transform = f64.Aff3{0, -1, h + minY, -1, 0, w + minX}
	case 8: // 逆时针旋转 90°
		transform = f64.Aff3{0, 1, -minY, -1, 0, w + minX}
	default:
		return src
	}
	width, height := bounds.Dx(), bounds.Dy()
	if orientation >= 5 {
		width, height = height, width
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.NearestNeighbor.Transform(dst, transform, src, bounds, draw.Src, nil)
	return dst
}

// webpFrameSize 返回 WebP 中 VP8/VP8L 帧的实际尺寸。VP8X 头声明的画布尺寸与帧尺寸相互独立，
// 解码器按帧尺寸分配内存，只校验画布尺寸会被 1x1 画布 + 超大帧的文件绕过。
func webpFrameSize(raw []byte) (int, int, error) {
	formType, chunks, err := riff.NewReader(bytes.NewReader(raw))
	if err != nil {
		return 0, 0, err
	}
	if formType != (riff.FourCC{'W', 'E', 'B', 'P'}) {
		return 0, 0, errors.New("not a webp container")
	}
	for {
		chunkID, chunkLen, chunkData, err := chunks.Next()
		if err != nil {
			return 0, 0, err
		}
		switch chunkID {
		case riff.FourCC{'V', 'P', '8', ' '}:
			decoder := vp8.NewDecoder()
			decoder.Init(chunkData, int(chunkLen))
			header, err := decoder.DecodeFrameHeader()
			return header.Width, header.Height, err
		case riff.FourCC{'V', 'P', '8', 'L'}:
			config, err := vp8l.DecodeConfig(chunkData)
			return config.Width, config.Height, err
		}
	}
}
