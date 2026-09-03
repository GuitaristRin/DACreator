package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GuitaristRin/DACreator/internal/model"
)

// 人工检查用：DAC_SAMPLE=1 go test ./internal/render/ -run TestManualCardOutput
func TestManualCardOutput(t *testing.T) {
	if os.Getenv("DAC_SAMPLE") == "" {
		t.Skip("设置 DAC_SAMPLE=1 以输出人工检查样张")
	}
	in := CardInput{
		PlayerID: "高橋リンタ", Region: "関東", City: "東京", Store: "ゲームセンター",
		Season: 5, Round: 4,
		Records: []model.Record{
			{Course: "秋名湖", Direction: "左周り", TimeMs: 147760, Rank: "LEGEND", National: "1"},
			{Course: "秋名", Direction: "下坡", TimeMs: 152300, Rank: "MASTER+", National: "3"},
			{Course: "妙義", Direction: "上坡", TimeMs: 168999, Rank: "EXPERT", National: "88"},
			{Course: "赤城", Direction: "逆时针", TimeMs: 185420, Rank: "EXPERT", National: "255"},
			{Course: "碓冰", Direction: "顺时针", TimeMs: 179010, Rank: "PROFESSIONAL", National: "331"},
			{Course: "椿线", Direction: "下坡", TimeMs: 199010, Rank: "PROFESSIONAL", National: "412"},
		},
		RoundScore: 612, PrideValue: 1288, TeamName: "Project D", TeamScore: 4581, TeamLevel: "GOLD",
	}
	img, err := RenderCard(in, DefaultCardConfig(filepath.Join("..", "..", "assets")))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(os.TempDir(), "dac_card_sample.png")
	if err := SavePNG(img, out); err != nil {
		t.Fatal(err)
	}
	t.Log("样张输出：", out)
}

// 人工检查用：DAC_SAMPLE=1 go test ./internal/render/ -run TestManualRecordCardOutput
func TestManualRecordCardOutput(t *testing.T) {
	if os.Getenv("DAC_SAMPLE") == "" {
		t.Skip("设置 DAC_SAMPLE=1 以输出人工检查样张")
	}
	records := []model.Record{
		{Course: "秋名湖", Direction: "左周り", TimeMs: 147760, Rank: "LEGEND", National: "1"},
		{Course: "秋名", Direction: "下坡", TimeMs: 152300, Rank: "MASTER+", National: "3"},
		{Course: "妙義", Direction: "上坡", TimeMs: 168999, Rank: "MASTER", National: "12"},
		{Course: "赤城", Direction: "逆时针", TimeMs: 185420, Rank: "EXPERT", National: "255"},
		{Course: "碓冰", Direction: "顺时针", TimeMs: 179010, Rank: "EXPERT", National: "331"},
		{Course: "椿线", Direction: "下坡", TimeMs: 199010, Rank: "EXPERT", National: "412"},
		{Course: "秋名雪", Direction: "下坡", TimeMs: 205123, Rank: "SPECIALIST", National: "12345"},
		{Course: "筑波", Direction: "去路", TimeMs: 158442, Rank: "SPECIALIST", National: "672"},
		{Course: "八方原", Direction: "归路", TimeMs: 214876, Rank: "ROOKIE", National: "20891"},
	}
	img, err := RenderRecordCard(RecordCardInput{
		PlayerID: "高橋リンタ", TeamName: "Project D", TeamLevel: "GOLD",
		Season: 5, Round: 4, RoundScore: 612, TeamScore: 4581, Records: records,
	}, DefaultRecordCardConfig(filepath.Join("..", "..", "assets")))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(os.TempDir(), "dac_record_card_sample.png")
	if err := SavePNG(img, out); err != nil {
		t.Fatal(err)
	}
	t.Log("样张输出：", out)
}
