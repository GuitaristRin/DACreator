// 记录卡：玩家 ID、所属车队、等级持有统计与精选计时赛成绩的单页卡片。
// 旧版曾有同名功能但实现遗失，此为按原要素的重制版。
// 与成绩卡共用同一张原画模板（头文字D 漫画底图 + 金属拉丝框），构图对应关系：
//
//	左列：玩家铭牌 → 车队铭牌 → 联赛等级铭牌 → 精选计时赛 Top5
//	右列：SEASON/ROUND → 回合分数 → 所属 TEAM → 等级徽章架（各类等级） → TEAM 分数
//
// 模板烙印的「回合分数 / 所属 TEAM / TEAM 分数」标签所辖区域均绘制语义匹配的内容。
package render

import (
	"fmt"
	"image"
	"image/draw"
	"path/filepath"

	"golang.org/x/image/font"

	"github.com/GuitaristRin/DACreator/internal/model"
)

// RecordCardInput 是渲染记录卡所需的全部输入。
type RecordCardInput struct {
	PlayerID   string
	TeamName   string
	TeamLevel  string
	Season     int
	Round      int
	RoundScore int
	TeamScore  int
	Records    []model.Record // 全部计时赛记录（等级统计取全集，精选区自动取 Top5）
}

// RecordCardConfig 记录卡渲染配置。
type RecordCardConfig struct {
	TemplatePath string
	FontPath     string
	RankImgDir   string
}

// DefaultRecordCardConfig 基于资产目录返回配置。
func DefaultRecordCardConfig(assetsDir string) RecordCardConfig {
	return RecordCardConfig{
		TemplatePath: filepath.Join(assetsDir, "render", "template.png"),
		FontPath:     filepath.Join(assetsDir, "font", "NotoSansCJKsc-Bold.otf"),
		RankImgDir:   filepath.Join(assetsDir, "rank"),
	}
}

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
	tpl, err := loadPNG(cfg.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("加载记录卡模板 %s: %w", cfg.TemplatePath, err)
	}
	canvas := image.NewRGBA(tpl.Bounds())
	draw.Draw(canvas, canvas.Bounds(), tpl, tpl.Bounds().Min, draw.Src)

	repo, err := newFontRepo(cfg.FontPath)
	if err != nil {
		return nil, err
	}
	faces := make(map[string]font.Face, 9)
	for name, size := range map[string]int{
		"player": 72, "location": 48, "store": 48, "track": 28,
		"season": 72, "score": 56, "team": 56, "teamscore": 48,
		"count": 30,
	} {
		f, err := repo.face(size)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		faces[name] = f
	}

	// ── 左列：铭牌三连 + 精选计时赛 Top5 ─────────────────────────────────
	cardText(canvas, faces["player"], cardPlayerIDArea, in.PlayerID, cardWhite, cardBlack, 3)
	cardText(canvas, faces["location"], cardLocationArea, in.TeamName, cardWhite, cardBlack, 2)
	league := ""
	if in.TeamName != "" && in.TeamLevel != "" {
		league = in.TeamLevel + " LEAGUE"
	}
	cardText(canvas, faces["store"], cardStoreArea, league, cardWhite, cardBlack, 2)

	renderFeaturedRecords(canvas, faces, in.Records, cfg)

	// ── 右列：赛季 / 回合分数 / 车队 / 等级徽章架 / 车队分数 ──────────────
	cardText(canvas, faces["season"], cardSeasonArea, fmt.Sprintf("SEASON %d ROUND %d", in.Season, in.Round), cardWhite, cardBlack, 3)
	drawScoreRight(canvas, faces["score"], cardScoreArea, fmt.Sprintf("%d pts", in.RoundScore))
	drawScoreRight(canvas, faces["team"], cardTeamNameArea, in.TeamName)

	renderRankTallyShelf(canvas, faces, in.Records, cfg)

	// TEAM 分数：模板烙印标签按成绩卡同位对齐，数值绘于标签正下方
	if in.TeamName != "" {
		x1, y1, _, y2 := cardHonorArea[0], cardHonorArea[1], cardHonorArea[2], cardHonorArea[3]
		badgeAreaH := int(float64(y2-y1) * 0.7)
		vx := x1 + cardHonorBadgeW/2 + cardTeamScoreXOff
		vy := y1 + (badgeAreaH-cardHonorBadgeH)/2 + cardHonorBadgeH + cardBadgeTextVOff + cardTeamScoreYOff
		drawStrokedTextMiddleTop(canvas, faces["teamscore"], vx, vy, fmt.Sprintf("%d pts", in.TeamScore), cardWhite, cardBlack, 2)
	}
	return canvas, nil
}

// renderFeaturedRecords 在左下大区绘制精选计时赛 Top5，行构图与成绩卡一致：
// 等级徽章 + 「赛道 方向  时间」，宽度足够时自适应追加全国排名。
func renderFeaturedRecords(canvas *image.RGBA, faces map[string]font.Face, records []model.Record, cfg RecordCardConfig) {
	x1, y1 := cardRecordsArea[0], cardRecordsArea[1]
	const (
		badgeW    = 220
		badgeH    = 56
		spacing   = 48
		textAvail = 440 // 徽章右侧的文本区宽度
	)
	top := SelectTopRecords(records, 5)
	for i, rec := range top {
		rowY := y1 + 40 + i*(badgeH+spacing)
		textX := x1 + 20
		if b := scaledBadge(cfg.RankImgDir, cardRankImageName(rec.Rank), badgeW, badgeH); b != nil {
			draw.Draw(canvas, image.Rect(textX, rowY, textX+b.Bounds().Dx(), rowY+b.Bounds().Dy()), b, b.Bounds().Min, draw.Over)
			textX += b.Bounds().Dx() + 20
		} else {
			_, th := textSize(faces["track"], rec.Rank)
			drawStrokedTextAt(canvas, faces["track"], textX, rowY+(badgeH-th)/2, rec.Rank, cardBlack, cardWhite, 1)
			textX += 250
		}

		track := truncateRunes(rec.Course+" "+rec.Direction, 25)
		full := fmt.Sprintf("%s  %s", track, model.FormatRaceTime(rec.TimeMs))
		if rec.National != "" {
			// 宽度足够才追加全国排名，避免挤压换行或越界
			withNat := fmt.Sprintf("%s  全国%s位", full, rec.National)
			if font.MeasureString(faces["track"], withNat).Ceil() <= textAvail {
				full = withNat
			}
		}
		textY := rowY + badgeH/2 - 15
		drawStrokedTextAt(canvas, faces["track"], textX, textY, full, cardBlack, cardWhite, 1)
	}
}

// renderRankTallyShelf 在右下荣誉区上部绘制等级徽章架：每行最多 4 枚，
// 徽章居中、持有数居中于徽章下方；行区避让模板烙印的「TEAM 分数」标签。
func renderRankTallyShelf(canvas *image.RGBA, faces map[string]font.Face, records []model.Record, cfg RecordCardConfig) {
	x1, y1, x2 := cardHonorArea[0], cardHonorArea[1], cardHonorArea[2]
	const (
		cols     = 4
		badgeW   = 170
		badgeH   = 48
		cellH    = 140
		shelfTop = 95 // 距荣誉区顶（避开所属 TEAM 行）
	)
	tally := RankTally(records)
	if len(tally) == 0 {
		drawStrokedTextAt(canvas, faces["track"], x1+60, y1+shelfTop+10, "暂无成绩记录", cardBlack, cardWhite, 1)
		return
	}
	cellW := (x2 - x1 - 40) / cols
	for i, rc := range tally {
		cx := x1 + 20 + (i%cols)*cellW
		cy := y1 + shelfTop + (i/cols)*cellH
		label := fmt.Sprintf("× %d", rc.Count)
		if b := scaledBadge(cfg.RankImgDir, cardRankImageName(rc.Rank), badgeW, badgeH); b != nil {
			bx := cx + (cellW-b.Bounds().Dx())/2
			draw.Draw(canvas, image.Rect(bx, cy, bx+b.Bounds().Dx(), cy+b.Bounds().Dy()), b, b.Bounds().Min, draw.Over)
			lw, lh := textSize(faces["count"], label)
			drawStrokedTextAt(canvas, faces["count"], cx+(cellW-lw)/2, cy+badgeH+(44-lh)/2, label, cardBlack, cardWhite, 1)
		} else {
			// 徽章缺失时以等级名文字兜底
			rw, _ := textSize(faces["track"], rc.Rank)
			drawStrokedTextAt(canvas, faces["track"], cx+(cellW-rw)/2, cy+8, rc.Rank, cardBlack, cardWhite, 1)
			lw, lh := textSize(faces["count"], label)
			drawStrokedTextAt(canvas, faces["count"], cx+(cellW-lw)/2, cy+badgeH+(44-lh)/2, label, cardBlack, cardWhite, 1)
		}
	}
}
