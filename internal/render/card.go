// 成绩卡渲染：基于 template.png 模板填充数据。
// 坐标与描边参数忠实移植旧版 render.py（含其中两处 y2<y1 的区域定义，保持输出位置一致）。
package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/GuitaristRin/DACreator/internal/model"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// CardInput 是渲染成绩卡所需的全部输入。
type CardInput struct {
	PlayerID string
	Region   string
	City     string
	Store    string
	Season   int
	Round    int

	Records    []model.Record // 全部计时赛记录（渲染前自动选取 Top5）
	RoundScore int
	PrideValue int
	TeamName   string
	TeamScore  int
	TeamLevel  string
}

// CardConfig 成绩卡渲染配置。
type CardConfig struct {
	TemplatePath string
	FontPath     string
	RankImgDir   string
	TeamImgDir   string
	PrideImgDir  string
}

// DefaultCardConfig 基于资产目录返回配置。
func DefaultCardConfig(assetsDir string) CardConfig {
	return CardConfig{
		TemplatePath: filepath.Join(assetsDir, "render", "template.png"),
		FontPath:     filepath.Join(assetsDir, "font", "NotoSansCJKsc-Bold.otf"),
		RankImgDir:   filepath.Join(assetsDir, "rank"),
		TeamImgDir:   filepath.Join(assetsDir, "team"),
		PrideImgDir:  filepath.Join(assetsDir, "pride"),
	}
}

// 模板区域（与旧版 render.py 一致；LOCATION/STORE 两处 y2<y1 为旧版原样，保持文字落点不变）。
var (
	cardPlayerIDArea = area(35, 250, 760, 325)
	cardLocationArea = area(35, 420, 760, 415)
	cardStoreArea    = area(35, 545, 760, 505)
	cardRecordsArea  = area(40, 580, 760, 1160)
	cardSeasonArea   = area(700, 210, 1560, 325)
	cardScoreArea    = area(760, 395, 1560, 490)
	cardTeamNameArea = area(760, 510, 1560, 570)
	cardHonorArea    = area(760, 505, 1560, 1160)
)

const (
	cardRecordStartX   = 60
	cardRecordStartY   = 40
	cardRecordSpacing  = 38
	cardBadgeWidth     = 220
	cardBadgeHeight    = 56
	cardBadgeTextGap   = 20
	cardHonorBadgeW    = 340
	cardHonorBadgeH    = 204
	cardHonorBadgeGap  = 100
	cardBadgeTextVOff  = 35
	cardTeamScoreXOff  = 60
	cardTeamScoreYOff  = 105
	cardPrideScoreYOff = -10
	cardScoreRight     = 60
)

var cardWhite = color.RGBA{255, 255, 255, 255}
var cardBlack = color.RGBA{0, 0, 0, 255}

// rankOrder 卡片选取记录时的等级顺序。
var rankOrder = []string{"ROOKIE", "REGULAR", "SPECIALIST", "EXPERT", "PROFESSIONAL", "MASTER", "MASTER+", "LEGEND"}

var teamImageMap = map[string]string{
	"OPEN": "open.png", "BASIC": "basic.png", "BRONZE": "bronze.png", "SILVER": "silver.png",
	"GOLD": "gold.png", "PLATINUM": "platinum.png", "MASTER": "master.png",
}

// SelectTopRecords 选取用于成绩卡的精选记录：按全国排名升序取前 limit 条。
// 等级阈值不随赛道难度校准——易赛道 PRO 随手可得，难赛道不研究到高熟练度连
// EXPERT 都难拿——因此等级不能作为跨赛道的第一排序键；全国排名才是难度与
// 竞争共同决定的相对位置，等级仅作同排名时的次级键，时间作最后兜底。
// 缺失全国排名的记录（旧 6 列 CSV）排在末尾，组内按等级从高到低、时间升序。
// （旧版超额时随机抽样，v3 改为取前 N 条以保证可复现。）
func SelectTopRecords(records []model.Record, limit int) []model.Record {
	rankVal := func(r model.Record) int {
		for i, name := range rankOrder {
			if name == r.Rank {
				return i
			}
		}
		return -1
	}
	sorted := make([]model.Record, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		ni, nj := nationalKey(sorted[i]), nationalKey(sorted[j])
		if ni != nj {
			return ni < nj
		}
		if vi, vj := rankVal(sorted[i]), rankVal(sorted[j]); vi != vj {
			return vi > vj
		}
		return sorted[i].TimeMs < sorted[j].TimeMs
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

// nationalKey 把全国排名转为可比较的整数键：取数字前缀（兼容「255位」写法），
// 缺失或无法解析时排在所有有效排名之后。
func nationalKey(r model.Record) int {
	s := r.National
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return math.MaxInt
	}
	n, err := strconv.Atoi(s[:end])
	if err != nil {
		return math.MaxInt
	}
	return n
}

// RenderCard 渲染成绩卡并返回图片。
func RenderCard(in CardInput, cfg CardConfig) (image.Image, error) {
	tpl, err := loadPNG(cfg.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("加载成绩卡模板 %s: %w", cfg.TemplatePath, err)
	}
	canvas := image.NewRGBA(tpl.Bounds())
	xdraw.Draw(canvas, canvas.Bounds(), tpl, tpl.Bounds().Min, xdraw.Src)

	// 单次解析字体，各角色尺寸复用（与旧版一致）
	repo, err := newFontRepo(cfg.FontPath)
	if err != nil {
		return nil, err
	}
	faces := make(map[string]font.Face, 9)
	for name, size := range map[string]int{
		"player": 72, "location": 48, "store": 48, "track": 28,
		"season": 72, "score": 56, "team": 56, "teamscore": 48, "pride": 38,
	} {
		f, err := repo.face(size)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		faces[name] = f
	}

	cardText(canvas, faces["player"], cardPlayerIDArea, in.PlayerID, cardWhite, cardBlack, 3)
	cardText(canvas, faces["location"], cardLocationArea, in.Region+"   "+in.City, cardWhite, cardBlack, 2)
	cardText(canvas, faces["store"], cardStoreArea, in.Store, cardWhite, cardBlack, 2)
	if err := renderCardRecords(canvas, faces["track"], in.Records, cfg); err != nil {
		return nil, err
	}

	cardText(canvas, faces["season"], cardSeasonArea, fmt.Sprintf("SEASON %d ROUND %d", in.Season, in.Round), cardWhite, cardBlack, 3)
	drawScoreRight(canvas, faces["score"], cardScoreArea, fmt.Sprintf("%d pts", in.RoundScore))
	drawScoreRight(canvas, faces["team"], cardTeamNameArea, in.TeamName)
	if err := renderHonorArea(canvas, faces, in, cfg); err != nil {
		return nil, err
	}
	return canvas, nil
}

// renderCardRecords 渲染左列计时赛 Top5。
func renderCardRecords(canvas *image.RGBA, face font.Face, records []model.Record, cfg CardConfig) error {
	top := SelectTopRecords(records, 5)
	x1, y1 := cardRecordsArea[0], cardRecordsArea[1]
	startX, startY := x1+cardRecordStartX, y1+cardRecordStartY

	for i, rec := range top {
		recordY := startY + i*(cardBadgeHeight+cardRecordSpacing)
		textX := startX
		if badge := scaledBadge(cfg.RankImgDir, cardRankImageName(rec.Rank), cardBadgeWidth, cardBadgeHeight); badge != nil {
			draw.Draw(canvas, image.Rect(startX, recordY, startX+badge.Bounds().Dx(), recordY+badge.Bounds().Dy()),
				badge, badge.Bounds().Min, draw.Over)
			textX = startX + cardBadgeWidth + cardBadgeTextGap
		}
		track := truncateRunes(rec.Course+" "+rec.Direction, 25)
		full := fmt.Sprintf("%s  %s", track, model.FormatRaceTime(rec.TimeMs))
		textY := recordY + cardBadgeHeight/2 - 15
		drawStrokedTextAt(canvas, face, textX, textY, full, cardBlack, cardWhite, 1)
	}
	return nil
}

// renderHonorArea 渲染右下车队/名声荣誉区。
func renderHonorArea(canvas *image.RGBA, faces map[string]font.Face, in CardInput, cfg CardConfig) error {
	x1, y1, x2, y2 := cardHonorArea[0], cardHonorArea[1], cardHonorArea[2], cardHonorArea[3]
	badgeAreaH := int(float64(y2-y1) * 0.7)
	totalW := cardHonorBadgeW*2 + cardHonorBadgeGap
	startX := x1 + (x2-x1-totalW)/2

	// 车队等级徽章
	teamImgName := teamImageMap[in.TeamLevel]
	if teamImgName == "" {
		teamImgName = "open.png"
	}
	if badge := scaledBadge(cfg.TeamImgDir, teamImgName, cardHonorBadgeW, cardHonorBadgeH); badge != nil {
		bx := startX
		by := y1 + (badgeAreaH-badge.Bounds().Dy())/2
		draw.Draw(canvas, image.Rect(bx, by, bx+badge.Bounds().Dx(), by+badge.Bounds().Dy()),
			badge, badge.Bounds().Min, draw.Over)
		// 车队分数：徽章下方，中点偏移
		vx := bx + badge.Bounds().Dx()/2 + cardTeamScoreXOff
		vy := by + badge.Bounds().Dy() + cardBadgeTextVOff + cardTeamScoreYOff
		drawStrokedTextMiddleTop(canvas, faces["teamscore"], vx, vy, fmt.Sprintf("%d pts", in.TeamScore), cardWhite, cardBlack, 2)
	}

	// 名声徽章（分数直接叠在徽章中央）
	prideName := "pride1.png"
	switch {
	case in.PrideValue >= 1000:
		prideName = "pride3.png"
	case in.PrideValue >= 100:
		prideName = "pride2.png"
	}
	if badge0 := loadBadge(cfg.PrideImgDir, prideName); badge0 != nil {
		pride := image.NewRGBA(image.Rect(0, 0, cardHonorBadgeW, cardHonorBadgeH))
		scaleInto(pride, badge0)
		px := startX + cardHonorBadgeW + cardHonorBadgeGap
		py := y1 + (badgeAreaH-cardHonorBadgeH)/2
		draw.Draw(canvas, image.Rect(px, py, px+cardHonorBadgeW, py+cardHonorBadgeH), pride, pride.Bounds().Min, draw.Over)
		// 分数叠在徽章中央（纵向偏移与旧版一致）
		tw, th := textSize(faces["pride"], fmt.Sprint(in.PrideValue))
		tx := px + (cardHonorBadgeW-tw)/2
		ty := py + (cardHonorBadgeH-th)/2 + cardPrideScoreYOff
		drawStrokedTextAt(canvas, faces["pride"], tx, ty, fmt.Sprint(in.PrideValue), cardWhite, cardBlack, 2)
	}
	return nil
}

// loadBadge 读取徽章图片；不存在返回 nil（调用方跳过绘制）。
func loadBadge(dir, name string) image.Image {
	if name == "" {
		return nil
	}
	img, err := loadPNG(filepath.Join(dir, name))
	if err != nil {
		return nil
	}
	return img
}

// scaledBadge 保持宽比缩放徽章至不超过 maxW×maxH。
func scaledBadge(dir, name string, maxW, maxH int) image.Image {
	src := loadBadge(dir, name)
	if src == nil {
		return nil
	}
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw == 0 || sh == 0 {
		return nil
	}
	ratio := min(float64(maxW)/float64(sw), float64(maxH)/float64(sh))
	dw, dh := int(float64(sw)*ratio), int(float64(sh)*ratio)
	if dw <= 0 || dh <= 0 {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	scaleInto(dst, src)
	return dst
}

// scaleInto 将 src 缩放填满 dst。
func scaleInto(dst *image.RGBA, src image.Image) {
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
}

// ---------- 绘制辅助 ----------

type rect4 = [4]int

func area(x1, y1, x2, y2 int) rect4 { return rect4{x1, y1, x2, y2} }

// cardText 在区域内居中绘制带描边文本（复刻旧版整数运算，含负高度区域的落点）。
func cardText(dst *image.RGBA, face font.Face, area rect4, text string, fill, stroke color.Color, strokeW int) {
	x1, y1, x2, y2 := area[0], area[1], area[2], area[3]
	w, h := textSize(face, text)
	x := x1 + (x2-x1-w)/2
	y := y1 + (y2-y1-h)/2
	drawStrokedTextAt(dst, face, x, y, text, fill, stroke, strokeW)
}

func drawScoreRight(dst *image.RGBA, face font.Face, area rect4, text string) {
	_, y1, x2, y2 := area[0], area[1], area[2], area[3]
	w, h := textSize(face, text)
	x := x2 - cardScoreRight - w
	y := y1 + (y2-y1-h)/2
	drawStrokedTextAt(dst, face, x, y, text, cardWhite, cardBlack, 2)
}

// drawStrokedTextAt 以 (x,y) 为文本左上角绘制描边文字。
func drawStrokedTextAt(dst *image.RGBA, face font.Face, x, y int, text string, fill, stroke color.Color, strokeW int) {
	if text == "" {
		return
	}
	metrics := face.Metrics()
	baseline := y + metrics.Ascent.Ceil()
	for dx := -strokeW; dx <= strokeW; dx++ {
		for dy := -strokeW; dy <= strokeW; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			drawFaceText(dst, face, x+dx, baseline+dy, text, stroke)
		}
	}
	drawFaceText(dst, face, x, baseline, text, fill)
}

// drawStrokedTextMiddleTop 以 (x,y) 为中上锚点绘制描边文字（旧版 anchor="mt"）。
func drawStrokedTextMiddleTop(dst *image.RGBA, face font.Face, x, y int, text string, fill, stroke color.Color, strokeW int) {
	w, _ := textSize(face, text)
	drawStrokedTextAt(dst, face, x-w/2, y, text, fill, stroke, strokeW)
}

func drawFaceText(dst *image.RGBA, face font.Face, x, baseline int, text string, c color.Color) {
	d := &font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, baseline)}
	d.DrawString(text)
}

func textSize(face font.Face, text string) (int, int) {
	w := font.MeasureString(face, text).Ceil()
	m := face.Metrics()
	return w, m.Ascent.Ceil() + m.Descent.Ceil()
}

// cardRankImageName 等级 → 徽章文件名。
func cardRankImageName(rank string) string {
	if rank == "MASTER+" {
		return "masterp.png"
	}
	for _, name := range rankOrder {
		if name == rank {
			return name + ".png" //nolint:gosec // 文件名来自固定映射
		}
	}
	return ""
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
