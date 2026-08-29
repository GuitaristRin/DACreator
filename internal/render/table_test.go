package render

import (
	"os"
	"path/filepath"
	"testing"

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
