package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GuitaristRin/DACreator/internal/model"
)

// 人工检查用：DAC_SAMPLE=1 go test ./internal/render/ -run TestManualSampleOutput
func TestManualSampleOutput(t *testing.T) {
	if os.Getenv("DAC_SAMPLE") == "" {
		t.Skip("设置 DAC_SAMPLE=1 以输出人工检查样张")
	}
	records := []model.Record{
		{Course: "秋名湖", Direction: "左周り", TimeMs: 147760, Rank: "EXPERT", Car: "CIVIC TYPE R (FL5) [HC]", National: "255", Date: "2026/01/19"},
		{Course: "秋名", Direction: "下坡", TimeMs: 152300, Rank: "LEGEND", Car: "AE86", National: "1", Date: "2025/12/21"},
		{Course: "妙義", Direction: "上坡", TimeMs: 168999, Rank: "MASTER+", Car: "RX-7 (FD3S)", National: "12", Date: "2025/11/02"},
		{Course: "赤城", Direction: "逆时针", TimeMs: 185420, Rank: "ROOKIE", Car: "SILVIA S15", National: "9803", Date: "2026/03/15"},
		{Course: "碓冰", Direction: "顺时针", TimeMs: 179010, Rank: "PROFESSIONAL", Car: "GT-R R34", National: "331", Date: "2026/02/08"},
	}
	cfg := DefaultConfig(filepath.Join("..", "..", "assets"))
	img, err := RenderTable(records, cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(os.TempDir(), "dac_table_sample.png")
	if err := SavePNG(img, out); err != nil {
		t.Fatal(err)
	}
	t.Log("样张输出：", out)
}
