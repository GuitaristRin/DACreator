package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font"

	"github.com/GuitaristRin/DACreator/internal/model"
)

func testAssetsDir(t *testing.T) string {
	t.Helper()
	// 仓库根的 assets 目录
	dir, err := filepath.Abs(filepath.Join("..", "..", "assets"))
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Fatalf("测试资产目录不存在：%s", dir)
	}
	return dir
}

func sampleRecords() []model.Record {
	return []model.Record{
		{Course: "秋名湖", Direction: "左周り", TimeMs: 147760, Rank: "EXPERT", Car: "CIVIC TYPE R (FL5) [HC]", National: "255", Date: "2026/01/19"},
		{Course: "秋名", Direction: "下坡", TimeMs: 152300, Rank: "LEGEND", Car: "AE86", National: "1", Date: "2025/12/21"},
		{Course: "妙義", Direction: "上坡", TimeMs: 168999, Rank: "未知评价", Car: "未知车型", National: "", Date: "2025/11/02"},
	}
}

func TestRenderTableDimensions(t *testing.T) {
	cfg := DefaultConfig(testAssetsDir(t))
	records := sampleRecords()
	img, err := RenderTable(records, cfg)
	if err != nil {
		t.Fatalf("渲染失败：%v", err)
	}
	wantW := (80 + 60 + 80 + 100 + 280 + 90 + 80 + 20)
	wantH := (40 + len(records)*30 + 20)
	if img.Bounds().Dx() != wantW || img.Bounds().Dy() != wantH {
		t.Errorf("尺寸不符：got %dx%d, want %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), wantW, wantH)
	}
}

func TestRenderTableEmptyRecords(t *testing.T) {
	cfg := DefaultConfig(testAssetsDir(t))
	img, err := RenderTable(nil, cfg)
	if err != nil {
		t.Fatalf("空记录渲染失败：%v", err)
	}
	if img.Bounds().Dy() != 40+20 {
		t.Errorf("空记录高度不符：%d", img.Bounds().Dy())
	}
}

func TestRenderTableMissingBadgeDir(t *testing.T) {
	cfg := DefaultConfig(t.TempDir())
	if _, err := RenderTable(sampleRecords(), cfg); err == nil {
		t.Fatal("徽章目录缺失应报错")
	}
}

func TestRankTally(t *testing.T) {
	cases := []struct {
		name    string
		records []model.Record
		want    []RankCount
	}{
		{
			"空记录",
			nil,
			nil,
		},
		{
			"按等级从高到低且跳过零项",
			[]model.Record{
				{Rank: "ROOKIE"}, {Rank: "LEGEND"}, {Rank: "LEGEND"},
				{Rank: "EXPERT"}, {Rank: "ROOKIE"}, {Rank: "ROOKIE"},
			},
			[]RankCount{{"LEGEND", 2}, {"EXPERT", 1}, {"ROOKIE", 3}},
		},
		{
			"未知等级按首次出现顺序排在末尾",
			[]model.Record{
				{Rank: "未知评价"}, {Rank: "MASTER"}, {Rank: "未知评价"},
			},
			[]RankCount{{"MASTER", 1}, {"未知评价", 2}},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := RankTally(tt.records)
			if len(got) != len(tt.want) {
				t.Fatalf("数量不符：got %v，want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("第 %d 项不符：got %+v，want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSavePNG(t *testing.T) {
	cfg := DefaultConfig(testAssetsDir(t))
	img, err := RenderTable(sampleRecords(), cfg)
	if err != nil {
		t.Fatalf("渲染失败：%v", err)
	}
	path := filepath.Join(t.TempDir(), "out.png")
	if err := SavePNG(img, path); err != nil {
		t.Fatalf("保存失败：%v", err)
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() == 0 {
		t.Fatalf("输出文件无效：%v", err)
	}
}

func TestFitText(t *testing.T) {
	face, err := newFontRepo(filepath.Join(testAssetsDir(t), "font", "NotoSansCJKsc-Bold.otf"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := face.face(24)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	measure := func(s string) int { return font.MeasureString(f, s).Ceil() }

	cases := []struct {
		name     string
		text     string
		maxWidth int
		want     string
	}{
		{"宽度足够原样返回", "AE86", 500, "AE86"},
		{"空串原样返回", "", 50, ""},
		{"恰好放下", "AE86", measure("AE86"), "AE86"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := fitText(tt.text, f, tt.maxWidth); got != tt.want {
				t.Errorf("fitText(%q, %d) = %q，期望 %q", tt.text, tt.maxWidth, got, tt.want)
			}
		})
	}

	t.Run("超长截断加省略号", func(t *testing.T) {
		const max = 200
		got := fitText("GT-R NISMO R35 [35GT-ABRR] 超长车型名测试", f, max)
		if !strings.HasSuffix(got, "…") {
			t.Fatalf("应以省略号结尾：%q", got)
		}
		if measure(got) > max {
			t.Errorf("截断后仍超宽：%q（%d > %d）", got, measure(got), max)
		}
		if got == "…" {
			t.Error("不应直接退化为单个省略号")
		}
	})

	t.Run("宽度极端不足", func(t *testing.T) {
		if got := fitText("AE86", f, measure("…")-1); got != "…" {
			t.Errorf("极端宽度应只剩省略号：%q", got)
		}
	})
}
