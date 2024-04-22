package module

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"

	"cube/internal/builtin"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/nfnt/resize"
	"golang.org/x/image/font/gofont/goregular"
)

func init() {
	register("image", func(worker Worker, db Db) interface{} {
		return &ImageClient{}
	})
}

type ImageClient struct{}

func (i *ImageClient) Create(width, height int) *Image {
	return &Image{gg.NewContext(width, height), 0}
}

func (i *ImageClient) Parse(input []byte) (*Image, error) {
	img, _, err := image.Decode(bytes.NewBuffer(input)) // 需要导入 "image/jpeg"、"image/png" 去解码 jpg、png 图片，否则当使用 image.Decode 处理图片文件时，会报 image: unknown format 错误
	if err != nil {
		return nil, err
	}

	return &Image{gg.NewContextForImage(img), 0}, nil
}

type Image struct {
	c        *gg.Context
	rotation float64 // 画布旋转角度数，用于定位底图坐标在旋转后的画布中的坐标位置
}

func (i *Image) Width() int {
	return i.c.Width()
}

func (i *Image) Height() int {
	return i.c.Height()
}

func (i *Image) Get(x int, y int) uint32 {
	r, g, b, a := i.c.Image().At(x, y).RGBA()
	return r << 24 & g << 16 & b << 8 & a
}

func (i *Image) Set(x int, y int, p uint32) {
	i.c.Image().(*image.RGBA).Set(x, y, color.RGBA{R: uint8(p >> 24), G: uint8(p >> 16), B: uint8(p >> 8), A: uint8(p)})
}

func (i *Image) SetDrawRotate(degrees float64) { // 旋转画布（但不会旋转底图），应在 DrawImage、DrawString 方法之前调用，用于绘画倾斜的图片或文字
	i.rotation = i.rotation + degrees
	i.c.Rotate(gg.Radians(degrees)) // 以 (0, 0) 为旋转中心，degrees 为顺时针旋转度数（如果是负数则表示逆时针旋转）
}

func (i *Image) SetDrawFontFace(fontSize float64, ttf []byte) error {
	if fontSize == 0 {
		fontSize = 15 // 字体大小，默认为 15
	}
	if len(ttf) == 0 {
		ttf = goregular.TTF // 字体，默认为 goregular.TTF
	}

	font, err := truetype.Parse(ttf)
	if err != nil {
		return err
	}

	face := truetype.NewFace(font, &truetype.Options{Size: fontSize})

	i.c.SetFontFace(face)

	return nil
}

func (i *Image) SetDrawColor(c interface{}) error {
	switch v := c.(type) {
	case string:
		i.c.SetHexColor(v)
	case []uint8:
		rgba := []uint8{0, 0, 0, 255}
		copy(rgba, v)
		i.c.SetColor(color.RGBA{rgba[0], rgba[1], rgba[2], rgba[3]})
	case []interface{}:
		rgba := []uint8{0, 0, 0, 255}
		for i, e := range v {
			rgba[i] = uint8(e.(int64))
		}
		i.c.SetColor(color.RGBA{rgba[0], rgba[1], rgba[2], rgba[3]})
	default:
		return errors.New("invalid color value")
	}
	return nil
}

func (i *Image) relocate(x, y float64) (float64, float64) {
	if i.rotation == 0 {
		return x, y
	}

	/**
	 * 画布以 (0, 0) 为旋转中心点顺时针旋转 rotation 度，即相当于相同坐标在数学平面直角坐标系中以原点为旋转中心顺时针旋转 rotation 度（如果 rotation 为负数则表示逆时针旋转）
	 * 假设底图中有坐标 (x, y)，其到原点的距离 c = Math.sqrt(x * x + y * y), 其与 x 轴的夹角为 t，那么 sin(t) = y / c, cos(t) = y / x
	 * 则底图中的坐标 (x, y) 对应到旋转后（旋转角为 a）的画布坐标 (x2, y2) 为
	 *     x2 = c * cos(a + t)
	 *        = c * (cos(a) * cos(t) - sin(a) * sin(t))
	 *        = c * (cos(a) * x / c - sin(a) * y / c)
	 *        = cos(a) * x - sin(a) * y
	 *     y2 = c * sin(a + t)
	 *        = c * (sin(a) * cos(t) + cos(a) * sin(t))
	 *        = c * (sin(a) * x / c + cos(a) * y / c)
	 *        = sin(a) * x + cos(a) * y
	 */
	sina, cosa := math.Sin(math.Pi/180*i.rotation*-1), math.Cos(math.Pi/180*i.rotation*-1) // 画布坐标系与数学平面直角坐标系的 y 轴方向是相反的，因此这里旋转角度都要乘 -1
	return cosa*x - sina*y, sina*x + cosa*y
}

func (i *Image) DrawImage(o *Image, x, y float64) {
	x, y = i.relocate(x, y)
	i.c.DrawImage(o.c.Image(), int(x), int(y))
}

func (i *Image) DrawString(s string, x, y, ax, ay, width, lineSpacing float64) {
	x, y = i.relocate(x, y)
	if width == 0 {
		i.c.DrawStringAnchored(s, x, y, ax, ay)
		return
	}
	if lineSpacing == 0 {
		lineSpacing = 1 // 行距，默认为 1.0 倍
	}
	i.c.DrawStringWrapped(s, x, y, ax, ay, width, lineSpacing, gg.AlignLeft)
}

func (i *Image) Resize(w uint, h uint) *Image {
	img := resize.Resize(w, h, i.c.Image(), resize.Bilinear)
	return &Image{gg.NewContextForImage(img), 0}
}

func (i *Image) ToJPG(quality int) (builtin.Buffer, error) {
	if quality == 0 {
		quality = 100 // 压缩质量因子，默认为 100
	}
	w := new(bytes.Buffer)
	if err := jpeg.Encode(w, i.c.Image(), &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (i *Image) ToPNG() (builtin.Buffer, error) {
	w := new(bytes.Buffer)
	if err := png.Encode(w, i.c.Image()); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
