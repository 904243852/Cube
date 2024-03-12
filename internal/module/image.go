package module

import (
	"bytes"
	"cube/internal/builtin"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg" // 需要导入 "image/jpeg"、"image/gif"、"image/png" 去解码 jpg、gif、png 图片，否则当使用 image.Decode 处理图片文件时，会报 image: unknown format 错误
	_ "image/png"
)

func init() {
	register("image", func(worker Worker, db Db) interface{} {
		return &ImageClient{}
	})
}

type ImageClient struct{}

func (i *ImageClient) Create(width int, height int) *ImageBuffer {
	return &ImageBuffer{
		image:   image.NewRGBA(image.Rect(0, 0, width, height)),
		Width:   width,
		offsetX: 0,
		Height:  height,
		offsetY: 0,
	}
}

func (i *ImageClient) Parse(input []byte) (*ImageBuffer, error) {
	img, _, err := image.Decode(bytes.NewBuffer(input)) // 图片文件解码
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	return &ImageBuffer{
		image:   img,
		Width:   bounds.Max.X - bounds.Min.X,
		offsetX: bounds.Min.X,
		Height:  bounds.Max.Y - bounds.Min.Y,
		offsetY: bounds.Min.Y,
	}, nil
}

type ImageBuffer struct {
	image   image.Image
	Width   int
	offsetX int
	Height  int
	offsetY int
}

func (i *ImageBuffer) Get(x int, y int) uint32 {
	r, g, b, a := i.image.At(x+i.offsetX, y+i.offsetY).RGBA()
	return r << 24 & g << 16 & b << 8 & a
}

func (i *ImageBuffer) Set(x int, y int, p uint32) {
	i.image.(*image.RGBA).Set(x+i.offsetX, y+i.offsetY, color.RGBA{R: uint8(p >> 24), G: uint8(p >> 16), B: uint8(p >> 8), A: uint8(p)})
}

func (i *ImageBuffer) ToBytes() (builtin.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, i.image, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i *ImageBuffer) Resize(w uint, h uint) *ImageBuffer {
	img := resize.Resize(w, h, i.image, resize.Bilinear)

	bounds := img.Bounds()

	return &ImageBuffer{
		image:   img,
		Width:   bounds.Max.X - bounds.Min.X,
		offsetX: bounds.Min.X,
		Height:  bounds.Max.Y - bounds.Min.Y,
		offsetY: bounds.Min.Y,
	}
}
