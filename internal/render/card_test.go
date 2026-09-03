package render

import (
	"strings"
	"testing"

	"github.com/GuitaristRin/DACreator/internal/model"
)

// wantKeys 以「等级/全国排名」序列描述期望的选取顺序。
func TestSelectTopRecords(t *testing.T) {
	r := func(rank, nat string, timeMs int) model.Record {
		return model.Record{Rank: rank, National: nat, TimeMs: timeMs}
	}
	cases := []struct {
		name    string
		records []model.Record
		limit   int
		want    []string
	}{
		{
			"全国排名优先于等级：秋名 EXPERT 胜过箱根 PRO",
			// 等级阈值不随赛道难度校准：箱根下 PRO 随手可得，秋名 EXPERT 需要深度研究
			[]model.Record{
				r("PROFESSIONAL", "800", 190000), // 箱根 下坡：易赛道，等级廉价
				r("EXPERT", "250", 168999),       // 秋名：难赛道，相对位置更硬
				r("EXPERT", "88", 152300),
			},
			3,
			[]string{"EXPERT/88", "EXPERT/250", "PROFESSIONAL/800"},
		},
		{
			"无全国排名者排末尾",
			[]model.Record{
				r("PROFESSIONAL", "", 150000),
				r("PROFESSIONAL", "50", 160000),
			},
			2,
			[]string{"PROFESSIONAL/50", "PROFESSIONAL/"},
		},
		{
			"高排名低等级胜过低排名高等级",
			[]model.Record{
				r("PROFESSIONAL", "1", 140000),
				r("LEGEND", "9000", 150000),
			},
			2,
			[]string{"PROFESSIONAL/1", "LEGEND/9000"},
		},
		{
			"按排名取前 limit 条",
			[]model.Record{
				r("LEGEND", "9000", 150000),
				r("EXPERT", "400", 168999),
				r("EXPERT", "12", 185420),
				r("EXPERT", "255", 147760),
				r("EXPERT", "331", 179010),
				r("EXPERT", "98", 176340),
				r("MASTER", "50", 160000),
			},
			5,
			[]string{"EXPERT/12", "MASTER/50", "EXPERT/98", "EXPERT/255", "EXPERT/331"},
		},
		{
			"全国排名相同按等级再按时间兜底",
			[]model.Record{
				r("EXPERT", "100", 152300),
				r("MASTER", "100", 160000),
				r("EXPERT", "100", 147760),
			},
			3,
			[]string{"MASTER/100", "EXPERT/100", "EXPERT/100"},
		},
		{
			"「255位」写法按数字解析",
			[]model.Record{
				r("PROFESSIONAL", "255位", 179010),
				r("PROFESSIONAL", "12", 199010),
			},
			2,
			[]string{"PROFESSIONAL/12", "PROFESSIONAL/255位"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectTopRecords(tt.records, tt.limit)
			if len(got) != len(tt.want) {
				t.Fatalf("选取数量不符：got %d 条，want %d 条", len(got), len(tt.want))
			}
			for i, key := range tt.want {
				wantRank, wantNat, _ := strings.Cut(key, "/")
				if got[i].Rank != wantRank || got[i].National != wantNat {
					t.Errorf("第 %d 条 = %s/%s，期望 %s/%s", i, got[i].Rank, got[i].National, wantRank, wantNat)
				}
			}
		})
	}
}
