package render

import (
	"image"
	"testing"

	"github.com/GuitaristRin/DACreator/internal/model"
)

// TestRecordCardStaysWithinFrame 回归测试：记录卡渲染不得在模板画框之外
// 留下任何像素（左框 x=70、右框 x=1534、顶部 y=208、底框 y=1148），
// 也不得跨过中央分隔线（x=688）。对应「徽章与文字出框」的既有问题。
func TestRecordCardStaysWithinFrame(t *testing.T) {
	img, err := RenderRecordCard(recordCardTestInput(), DefaultRecordCardConfig(testAssetsDir(t)))
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := loadPNG(testAssetsDir(t) + "/render/template.png")
	if err != nil {
		t.Fatal(err)
	}

	if img.Bounds() != tpl.Bounds() {
		t.Fatalf("渲染尺寸与模板不一致：%v vs %v", img.Bounds(), tpl.Bounds())
	}

	const tol = 40 // 亮度容差，避免模板本身的 JPEG 噪声误报
	bounds := img.Bounds()

	// 模板画框之外的一切像素必须与模板一致（我们什么都没画）
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			outside := x < 70 || x > 1534 || y < 208 || y > 1148
			if !outside {
				continue
			}
			if diff(px(img, x, y), px(tpl, x, y)) > tol {
				t.Fatalf("画框外出现绘制内容：(x=%d, y=%d) diff=%d", x, y, diff(px(img, x, y), px(tpl, x, y)))
			}
		}
	}

	// 中央分隔线两侧（688..700）不得有左列内容横穿
	for y := 350; y < 1100; y++ {
		for x := 690; x < 700; x++ {
			if diff(px(img, x, y), px(tpl, x, y)) > tol {
				t.Fatalf("内容横穿中央分隔线：(x=%d, y=%d)", x, y)
			}
		}
	}
}

func recordCardTestInput() RecordCardInput {
	return RecordCardInput{
		PlayerID:   "高橋リンタ",
		TeamName:   "Project D",
		TeamLevel:  "GOLD",
		Season:     5,
		Round:      4,
		RoundScore: 612,
		TeamScore:  4581,
		Records: []model.Record{
			{Course: "秋名湖", Direction: "左周り", TimeMs: 147760, Rank: "LEGEND", National: "1"},
			{Course: "秋名", Direction: "下坡", TimeMs: 152300, Rank: "MASTER+", National: "3"},
			{Course: "妙義", Direction: "上坡", TimeMs: 168999, Rank: "MASTER", National: "12"},
			{Course: "赤城", Direction: "逆时针", TimeMs: 185420, Rank: "EXPERT", National: "255"},
			{Course: "箱根", Direction: "下坡", TimeMs: 159010, Rank: "PROFESSIONAL", National: "400"},
			{Course: "椿线", Direction: "下坡", TimeMs: 199010, Rank: "EXPERT", National: "412"},
			{Course: "秋名雪", Direction: "下坡", TimeMs: 205123, Rank: "SPECIALIST", National: "12345"},
			{Course: "筑波", Direction: "去路", TimeMs: 158442, Rank: "SPECIALIST", National: "672"},
			{Course: "八方原", Direction: "归路", TimeMs: 214876, Rank: "ROOKIE", National: "20891"},
		},
	}
}

func px(img image.Image, x, y int) [4]uint32 {
	r, g, b, a := img.At(x, y).RGBA()
	return [4]uint32{r >> 8, g >> 8, b >> 8, a >> 8}
}

func diff(a, b [4]uint32) int {
	d := 0
	for i := 0; i < 4; i++ {
		v := int(a[i]) - int(b[i])
		if v < 0 {
			v = -v
		}
		if v > d {
			d = v
		}
	}
	return d
}
