// 记录卡：玩家 ID、所属车队、等级持有统计与精选计时赛成绩的单页卡片。
// 旧版曾有同名功能但实现遗失，此为按原要素的重制版；
// 无模板可用，与成绩表一样纯代码绘制（2x 超采样），配色与成绩表同族。
package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
	"time"

	"golang.org/x/image/font"

	"github.com/GuitaristRin/DACreator/internal/model"
)

// 记录卡画布尺寸（1x）。
const (
	recordCardW = 1560
	recordCardH = 1160
)

// RecordCardInput 是渲染记录卡所需的全部输入。
type RecordCardInput struct {
	PlayerID  string
	TeamName  string
	TeamLevel string
	Season    int
	Round     int
	Records   []model.Record // 全部计时赛记录（等级统计取全集，精选区自动取 Top5）
}

// RecordCardConfig 记录卡渲染配置。
type RecordCardConfig struct {
	FontPath   string
	RankImgDir string
	TeamImgDir string
}

// DefaultRecordCardConfig 基于资产目录返回配置。
func DefaultRecordCardConfig(assetsDir string) RecordCardConfig {
	return RecordCardConfig{
		FontPath:   filepath.Join(assetsDir, "font", "NotoSansCJKsc-Bold.otf"),
		RankImgDir: filepath.Join(assetsDir, "rank"),
		TeamImgDir: filepath.Join(assetsDir, "team"),
	}
}

// 卡片配色（与成绩表同族：藏青 + 白底黑字）。
var (
	recordCardNavy      = color.RGBA{44, 62, 80, 255}
	recordCardCaption   = color.RGBA{176, 192, 208, 255}
	recordCardSubtle    = color.RGBA{159, 176, 192, 255}
	recordCardText      = color.RGBA{30, 30, 30, 255}
	recordCardSecondary = color.RGBA{120, 120, 130, 255}
	recordCardSeparator = color.RGBA{224, 224, 224, 255}
)

// RankCount 是一个等级的持有数量。
type RankCount struct {
	Rank  string
	Count int
}

// RankTally 统计各等级的持有数量：已知等级从高到低排列（仅保留数量大于 0 的），
// 未收录的等级按首次出现顺序排在末尾。
func RankTally(records []model.Record) []RankCount {
	counts := make(map[string]int)
	known := make(map[string]bool, len(rankOrder))
	for _, name := range rankOrder {
		known[name] = true
	}
	var extras []string
	seen := make(map[string]bool)
	for _, r := range records {
		counts[r.Rank]++
		if !known[r.Rank] && !seen[r.Rank] {
			seen[r.Rank] = true
			extras = append(extras, r.Rank)
		}
	}
	out := make([]RankCount, 0, len(counts))
	for i := len(rankOrder) - 1; i >= 0; i-- {
		if n := counts[rankOrder[i]]; n > 0 {
			out = append(out, RankCount{Rank: rankOrder[i], Count: n})
		}
	}
	for _, name := range extras {
		out = append(out, RankCount{Rank: name, Count: counts[name]})
	}
	return out
}

// RenderRecordCard 渲染记录卡并返回图片。
func RenderRecordCard(in RecordCardInput, cfg RecordCardConfig) (image.Image, error) {
	const scale = 2
	px := func(v int) int { return v * scale }
	w, h := px(recordCardW), px(recordCardH)

	repo, err := newFontRepo(cfg.FontPath)
	if err != nil {
		return nil, err
	}
	faces := make(map[string]font.Face, 8)
	for name, size := range map[string]int{
		"caption": 28, "player": 68, "meta": 30, "section": 38,
		"tier": 44, "row": 34, "rowSub": 30, "footer": 24,
	} {
		f, err := repo.face(px(size))
		if err != nil {
			return nil, err
		}
		defer f.Close()
		faces[name] = f
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	d := &drawer{img: img}

	// 白底（画布零值为透明，直接绘制会得到黑底）
	fillRect(img, image.Rect(0, 0, w, h), color.White)

	// ── 头部：藏青底 + 玩家 ID + 车队徽章 ─────────────────────────────────
	fillRect(img, image.Rect(0, 0, w, px(230)), recordCardNavy)
	d.text("RECORD CARD", faces["caption"], px(60), px(40), px(36), recordCardCaption)
	d.text(fitText(in.PlayerID, faces["player"], px(1000)), faces["player"], px(60), px(84), px(80), color.White)
	d.text(fmt.Sprintf("SEASON %d · ROUND %d", in.Season, in.Round), faces["meta"], px(60), px(182), px(36), recordCardSubtle)

	if in.TeamName != "" {
		if badge := scaledBadge(cfg.TeamImgDir, teamImageMap[in.TeamLevel], px(200), px(120)); badge != nil {
			bw := badge.Bounds().Dx()
			bx := w - px(60) - bw
			draw.Draw(img, image.Rect(bx, px(34), bx+bw, px(34)+badge.Bounds().Dy()), badge, badge.Bounds().Min, draw.Over)
			nameW, _ := textSize(faces["row"], in.TeamName)
			d.text(fitText(in.TeamName, faces["row"], bw+px(120)), faces["row"], bx+(bw-nameW)/2, px(164), px(44), color.White)
		} else {
			d.textRight(in.TeamName, faces["row"], w-px(60), px(92), px(48), color.White)
		}
	}

	// ── 各类等级：徽章 × 持有数，最多每行 4 个 ───────────────────────────
	section := func(title string, y int) {
		fillRect(img, image.Rect(px(60), px(y), px(72), px(y+34)), recordCardNavy)
		d.text(title, faces["section"], px(88), px(y-6), px(46), recordCardNavy)
	}
	section("持有等级", 280)

	tally := RankTally(in.Records)
	if len(tally) == 0 {
		d.text("暂无成绩记录", faces["rowSub"], px(88), px(360), px(40), recordCardSecondary)
	} else {
		const cols = 4
		cellW := (recordCardW - 120) / cols
		for i, rc := range tally {
			cx := px(60 + (i%cols)*cellW)
			cy := px(356 + (i/cols)*116)
			if b := scaledBadge(cfg.RankImgDir, cardRankImageName(rc.Rank), px(230), px(64)); b != nil {
				draw.Draw(img, image.Rect(cx, cy, cx+b.Bounds().Dx(), cy+b.Bounds().Dy()), b, b.Bounds().Min, draw.Over)
				d.text(fmt.Sprintf("× %d", rc.Count), faces["tier"], cx+b.Bounds().Dx()+px(20), cy, b.Bounds().Dy(), recordCardText)
			} else {
				// 徽章缺失时以等级名文字兜底
				d.text(rc.Rank, faces["row"], cx, cy, px(64), recordCardText)
				d.text(fmt.Sprintf("× %d", rc.Count), faces["tier"], cx+px(250), cy, px(64), recordCardText)
			}
		}
	}

	// ── 精选计时赛：Top5 徽章 + 赛道 · 方向 + 时间 + 全国排名 ─────────────
	section("精选计时赛", 648)
	top := SelectTopRecords(in.Records, 5)
	const rowH = 80
	for i, rec := range top {
		cy := px(716 + i*rowH)
		textX := px(60)
		if b := scaledBadge(cfg.RankImgDir, cardRankImageName(rec.Rank), px(220), px(56)); b != nil {
			draw.Draw(img, image.Rect(textX, cy, textX+b.Bounds().Dx(), cy+b.Bounds().Dy()), b, b.Bounds().Min, draw.Over)
			textX += b.Bounds().Dx() + px(36)
		} else {
			d.text(rec.Rank, faces["row"], textX, cy, px(64), recordCardText)
			textX += px(250)
		}
		d.text(fitText(rec.Course+" · "+rec.Direction, faces["row"], px(430)), faces["row"], textX, cy, px(64), recordCardText)
		d.textRight(model.FormatRaceTime(rec.TimeMs), faces["row"], px(1180), cy, px(64), recordCardText)
		national := "—"
		if rec.National != "" {
			national = "全国 " + rec.National + "位"
		}
		d.textRight(national, faces["rowSub"], w-px(60), cy, px(64), recordCardSecondary)
		if i < len(top)-1 {
			fillRect(img, image.Rect(px(60), cy+px(72), w-px(60), cy+px(73)), recordCardSeparator)
		}
	}

	// ── 页脚 ────────────────────────────────────────────────────────────
	footer := fmt.Sprintf("Generated by DACreator · %s · arcadezone.cn", time.Now().Format("2006/01/02"))
	d.text(footer, faces["footer"], px(60), px(1106), px(32), recordCardSecondary)

	return downscale(img, recordCardW, recordCardH), nil
}
