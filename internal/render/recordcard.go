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
// 模板实测严格几何（1600×1200 画布）：
// 左外框 x=70，中央竖直分隔线 x=688，右外框 x=1534。
// 顶部标题栏下沿 y=208，卡片底框 y=1148。
// 任何绘制坐标必须严格落在这些框线内部，留足内边距，严禁压线或出界。
const (
	tplLeftFrame  = 70
	tplDividerX   = 688
	tplRightFrame = 1534
)

// plateText 在左列铭牌分区内居中绘制文本，超宽时按实测框线截断。
func plateText(canvas *image.RGBA, face font.Face, area rect4, text string, strokeW int) {
	// 左右各留 20px 安全边距
	maxW := (tplDividerX - 20) - (tplLeftFrame + 20)
	cardText(canvas, face, area, fitText(text, face, maxW), cardWhite, cardBlack, strokeW)
}

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
		"player": 68, "location": 46, "store": 44, "track": 24,
		"season": 72, "score": 56, "team": 56, "teamscore": 48,
		"count": 28,
	} {
		f, err := repo.face(size)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		faces[name] = f
	}

	// ── 左列：铭牌三连（实测框线 [70, 688] 居中对齐）─────────────────────
	platePlayerArea := area(tplLeftFrame, 208, tplDividerX, 368)
	plateTeamArea := area(tplLeftFrame, 368, tplDividerX, 478)
	plateLeagueArea := area(tplLeftFrame, 478, tplDividerX, 574)

	plateText(canvas, faces["player"], platePlayerArea, in.PlayerID, 3)
	plateText(canvas, faces["location"], plateTeamArea, in.TeamName, 2)
	league := ""
	if in.TeamName != "" && in.TeamLevel != "" {
		league = in.TeamLevel + " LEAGUE"
	}
	plateText(canvas, faces["store"], plateLeagueArea, league, 2)

	// 左下大区：精选计时赛 Top5（严格落在 [70, 688] 内部）
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

// renderFeaturedRecords 在左下大区绘制精选计时赛 Top5。
// 边界约束：左框 x=70，中隔线 x=688，可用宽度 618px。
// 徽章起点放在 x=95（距左框 25px 留白），徽章宽 190，右侧文本区留 365px，永不压中线。
func renderFeaturedRecords(canvas *image.RGBA, faces map[string]font.Face, records []model.Record, cfg RecordCardConfig) {
	const (
		startX     = 95  // 严格大于 tplLeftFrame(70)，留 25px 边距，徽章决不出框
		startY     = 615 // y=574 横线下方留 41px 边距
		badgeW     = 185
		badgeH     = 46
		rowPitch   = 100 // 5 行分布在 615..1115，底框在 y=1148
		textGap    = 16
		textRightX = tplDividerX - 20 // 668，距中线留 20px 安全间距
	)
	textStartX := startX + badgeW + textGap // 95 + 185 + 16 = 296
	textAvail := textRightX - textStartX    // 668 - 296 = 372px

	top := SelectTopRecords(records, 5)
	for i, rec := range top {
		rowY := startY + i*rowPitch
		if b := scaledBadge(cfg.RankImgDir, cardRankImageName(rec.Rank), badgeW, badgeH); b != nil {
			// 徽章在 185x46 格内居中
			bx := startX + (badgeW-b.Bounds().Dx())/2
			by := rowY + (badgeH-b.Bounds().Dy())/2
			draw.Draw(canvas, image.Rect(bx, by, bx+b.Bounds().Dx(), by+b.Bounds().Dy()), b, b.Bounds().Min, draw.Over)
		} else {
			_, th := textSize(faces["track"], rec.Rank)
			drawStrokedTextAt(canvas, faces["track"], startX, rowY+(badgeH-th)/2, rec.Rank, cardBlack, cardWhite, 1)
		}

		track := rec.Course + " " + rec.Direction
		timeStr := model.FormatRaceTime(rec.TimeMs)
		full := fmt.Sprintf("%s %s", track, timeStr)
		if rec.National != "" {
			withNat := fmt.Sprintf("%s 全国%s位", full, rec.National)
			if font.MeasureString(faces["track"], withNat).Ceil() <= textAvail {
				full = withNat
			}
		}

		_, th := textSize(faces["track"], full)
		textY := rowY + (badgeH-th)/2
		drawStrokedTextAt(canvas, faces["track"], textStartX, textY, fitText(full, faces["track"], textAvail), cardBlack, cardWhite, 1)
	}
}

// renderRankTallyShelf 在右下荣誉区绘制等级徽章架：4 列网格。
// 边界约束：中隔线 x=688，右外框 x=1534。
// 区域限定在 x=725 .. 1495（宽 770px，左右各留 ~38px 安全边距，徽章决不出框）。
func renderRankTallyShelf(canvas *image.RGBA, faces map[string]font.Face, records []model.Record, cfg RecordCardConfig) {
	const (
		gridLeft  = 735
		gridRight = 1490
		gridTop   = 600
		cols      = 4
		cellH     = 135
		badgeMaxW = 150
		badgeMaxH = 42
	)
	tally := RankTally(records)
	if len(tally) == 0 {
		drawStrokedTextAt(canvas, faces["track"], gridLeft+20, gridTop+20, "暂无成绩记录", cardBlack, cardWhite, 1)
		return
	}

	cellW := (gridRight - gridLeft) / cols // ~188px
	for i, rc := range tally {
		cx := gridLeft + (i%cols)*cellW
		cy := gridTop + (i/cols)*cellH
		label := fmt.Sprintf("× %d", rc.Count)

		if b := scaledBadge(cfg.RankImgDir, cardRankImageName(rc.Rank), badgeMaxW, badgeMaxH); b != nil {
			bx := cx + (cellW-b.Bounds().Dx())/2
			draw.Draw(canvas, image.Rect(bx, cy, bx+b.Bounds().Dx(), cy+b.Bounds().Dy()), b, b.Bounds().Min, draw.Over)
			lw, lh := textSize(faces["count"], label)
			drawStrokedTextAt(canvas, faces["count"], cx+(cellW-lw)/2, cy+badgeMaxH+(42-lh)/2, label, cardBlack, cardWhite, 1)
		} else {
			rw, _ := textSize(faces["track"], rc.Rank)
			drawStrokedTextAt(canvas, faces["track"], cx+(cellW-rw)/2, cy+4, rc.Rank, cardBlack, cardWhite, 1)
			lw, lh := textSize(faces["count"], label)
			drawStrokedTextAt(canvas, faces["count"], cx+(cellW-lw)/2, cy+badgeMaxH+(42-lh)/2, label, cardBlack, cardWhite, 1)
		}
	}
}
