// Package render 将成绩数据渲染为可视化表格图片（旧版 PIL 实现的 Go 重写）。
// 渲染策略与旧版一致：先以 2 倍尺寸绘制再缩小，获得抗锯齿的清晰输出。
package render

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/GuitaristRin/DACreator/internal/model"
)

// Config 是表格渲染的外观参数，数值与旧版保持一致。
type Config struct {
	FontPath     string // CJK 粗体字体（Noto Sans CJK）
	RankImgDir   string // 等级徽章图片目录
	FontSize     int
	HeaderHeight int
	RowHeight    int
	ColWidths    []int // 7 列宽度
	Scale        int

	BgColor         color.Color
	HeaderColor     color.Color
	HeaderTextColor color.Color
	RowEvenColor    color.Color
	RowOddColor     color.Color
	TextColor       color.Color
	BorderColor     color.Color

	RankImgScale float64 // 徽章高度占行高的比例
	RankImgFiles map[string]string
}

// DefaultConfig 基于资产目录返回默认配置。
func DefaultConfig(assetsDir string) Config {
	return Config{
		FontPath:     filepath.Join(assetsDir, "font", "NotoSansCJKsc-Bold.otf"),
		RankImgDir:   filepath.Join(assetsDir, "rank"),
		FontSize:     12,
		HeaderHeight: 40,
		RowHeight:    30,
		ColWidths:    []int{80, 60, 80, 100, 280, 90, 80},
		Scale:        2,

		BgColor:         color.RGBA{255, 255, 255, 255},
		HeaderColor:     color.RGBA{44, 62, 80, 255},
		HeaderTextColor: color.RGBA{255, 255, 255, 255},
		RowEvenColor:    color.RGBA{245, 245, 245, 255},
		RowOddColor:     color.RGBA{255, 255, 255, 255},
		TextColor:       color.RGBA{0, 0, 0, 255},
		BorderColor:     color.RGBA{200, 200, 200, 255},

		RankImgScale: 0.8,
		RankImgFiles: RankImgFiles,
	}
}

// RankImgFiles 等级 → 徽章文件名（与旧版一致）。
var RankImgFiles = map[string]string{
	"ROOKIE":       "rookie.png",
	"REGULAR":      "regular.png",
	"SPECIALIST":   "specialist.png",
	"EXPERT":       "expert.png",
	"PROFESSIONAL": "professional.png",
	"MASTER":       "master.png",
	"MASTER+":      "masterp.png",
	"LEGEND":       "legend.png",
}

// RenderTable 渲染成绩表格图片。
func RenderTable(records []model.Record, cfg Config) (image.Image, error) {
	face, err := newFace(cfg.FontPath, cfg.FontSize*cfg.Scale)
	if err != nil {
		return nil, err
	}
	defer face.Close()
	headerFace, err := newFace(cfg.FontPath, 14*cfg.Scale)
	if err != nil {
		return nil, err
	}
	defer headerFace.Close()

	badges, err := newBadgeCache(cfg)
	if err != nil {
		return nil, err
	}

	colWidths := cfg.ColWidths
	if len(colWidths) != len(model.Columns) {
		return nil, fmt.Errorf("列宽配置需为 %d 列，实际 %d 列", len(model.Columns), len(colWidths))
	}
	scale := cfg.Scale
	margin := 10 * scale
	totalW := (sum(colWidths) + 20) * scale
	totalH := (cfg.HeaderHeight + len(records)*cfg.RowHeight + 20) * scale

	img := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	d := &drawer{img: img}

	// 背景
	fillRect(img, img.Bounds(), cfg.BgColor)

	// 表头
	y := margin
	fillRect(img, image.Rect(margin, y, totalW-margin, y+cfg.HeaderHeight*scale), cfg.HeaderColor)
	outlineRect(img, image.Rect(margin, y, totalW-margin, y+cfg.HeaderHeight*scale), cfg.BorderColor)
	x := margin
	for i, col := range model.Columns {
		d.text(col, headerFace, x+5*scale, y, cfg.HeaderHeight*scale, cfg.HeaderTextColor)
		x += colWidths[i] * scale
	}
	y += cfg.HeaderHeight * scale

	// 数据行
	for idx, rec := range records {
		rowColor := cfg.RowEvenColor
		if idx%2 == 1 {
			rowColor = cfg.RowOddColor
		}
		fillRect(img, image.Rect(margin, y, totalW-margin, y+cfg.RowHeight*scale), rowColor)
		outlineRect(img, image.Rect(margin, y, totalW-margin, y+cfg.RowHeight*scale), cfg.BorderColor)

		x = margin
		row := []struct {
			text  string
			badge string
		}{
			{rec.Course, ""},
			{rec.Direction, ""},
			{model.FormatRaceTime(rec.TimeMs), ""},
			{"", rec.Rank},
			{rec.Car, ""},
			{rec.National, ""},
			{rec.Date, ""},
		}
		for i, cell := range row {
			if cell.badge != "" {
				if b := badges.get(cell.badge, cfg); b != nil {
					drawCentered(img, b, image.Rect(x, y, x+colWidths[i]*scale, y+cfg.RowHeight*scale))
				} else {
					d.text(cell.badge, face, x+5*scale, y, cfg.RowHeight*scale, cfg.TextColor)
				}
			} else {
				d.text(cell.text, face, x+5*scale, y, cfg.RowHeight*scale, cfg.TextColor)
			}
			x += colWidths[i] * scale
		}
		y += cfg.RowHeight * scale
	}

	// 缩小回目标尺寸（等价旧版 2x 超采样）
	return downscale(img, totalW/scale, totalH/scale), nil
}

// SavePNG 将图片写入文件。
func SavePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建图片文件 %s: %w", path, err)
	}
	defer f.Close()
	if err := pngEncoder.Encode(f, img); err != nil {
		return fmt.Errorf("编码 PNG %s: %w", path, err)
	}
	return nil
}

func sum(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}
