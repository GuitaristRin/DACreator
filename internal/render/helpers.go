package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var pngEncoder = png.Encoder{}

// newFace 解析字体文件并创建指定像素大小的字型。
func newFace(path string, pixels int) (font.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取字体 %s: %w", path, err)
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("解析字体 %s: %w", path, err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(pixels),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("创建字型 %s: %w", path, err)
	}
	return face, nil
}

// drawer 在 image 上以垂直居中方式绘制文本。
type drawer struct {
	img *image.RGBA
}

func (d *drawer) text(s string, face font.Face, x, cellTop, cellH int, c color.Color) {
	if s == "" {
		return
	}
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	baseline := cellTop + (cellH+ascent-descent)/2

	fd := &font.Drawer{
		Dst:  d.img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, baseline),
	}
	fd.DrawString(s)
}

// badgeCache 按需加载并缩放等级徽章。
type badgeCache struct {
	cfg    Config
	loaded map[string]image.Image
}

func newBadgeCache(cfg Config) (*badgeCache, error) {
	if st, err := os.Stat(cfg.RankImgDir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("等级徽章目录不存在：%s", cfg.RankImgDir)
	}
	return &badgeCache{cfg: cfg, loaded: map[string]image.Image{}}, nil
}

func (b *badgeCache) get(rank string, cfg Config) image.Image {
	if img, ok := b.loaded[rank]; ok {
		return img
	}
	name, ok := cfg.RankImgFiles[rank]
	if !ok {
		return nil
	}
	img, err := loadPNG(filepath.Join(cfg.RankImgDir, name))
	b.loaded[rank] = img // 失败也缓存 nil，避免反复读盘
	if err != nil {
		return nil
	}

	// 目标高度 = 行高 × 比例（乘 Scale，因为整体在 2x 画布上绘制）
	targetH := int(float64(cfg.RowHeight*cfg.Scale) * cfg.RankImgScale)
	if targetH <= 0 || img.Bounds().Dy() == 0 {
		return nil
	}
	targetW := img.Bounds().Dx() * targetH / img.Bounds().Dy()
	if targetW <= 0 {
		return nil
	}
	scaled := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), xdraw.Over, nil)
	b.loaded[rank] = scaled
	return scaled
}

func loadPNG(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解析徽章 %s: %w", path, err)
	}
	return img, nil
}

// drawCentered 将 src 居中绘制到 cell 矩形内。
func drawCentered(dst *image.RGBA, src image.Image, cell image.Rectangle) {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	x := cell.Min.X + (cell.Dx()-sw)/2
	y := cell.Min.Y + (cell.Dy()-sh)/2
	xdraw.Draw(dst, image.Rect(x, y, x+sw, y+sh), src, src.Bounds().Min, xdraw.Over)
}

func fillRect(dst *image.RGBA, r image.Rectangle, c color.Color) {
	xdraw.Draw(dst, r, image.NewUniform(c), image.Point{}, xdraw.Src)
}

func outlineRect(dst *image.RGBA, r image.Rectangle, c color.Color) {
	// 1px（物理像素）描边：上、下、左、右
	for x := r.Min.X; x < r.Max.X; x++ {
		dst.Set(x, r.Min.Y, c)
		dst.Set(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		dst.Set(r.Min.X, y, c)
		dst.Set(r.Max.X-1, y, c)
	}
}

// downscale 用 Catmull-Rom 缩小图片（等价旧版 LANCZOS 的抗锯齿效果）。
func downscale(src *image.RGBA, w, h int) image.Image {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(out, out.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return out
}
