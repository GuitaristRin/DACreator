package az

import "testing"

// 与真实排行榜页面同构的回合映射内嵌样例（Season 5 起跳号为官方真实数据）。
const roundsFixtureHTML = `<!doctype html><html><head><meta name="csrf-token" content="tok"></head><body>
<script>window.roundsBySeason = {"4":[{"id":4,"round_event_nm":"Arcade Zone Season4 Round 1"},{"id":6,"round_event_nm":"Arcade Zone Season4 Round 2"}],"5":[{"id":5,"round_event_nm":"Arcade Zone Season5 Round 1"},{"id":7,"round_event_nm":"Arcade Zone Season5 Round 2"},{"id":8,"round_event_nm":"Arcade Zone Season5 Round 3"},{"id":9,"round_event_nm":"Arcade Zone Season5 Round 4"},{"id":10,"round_event_nm":"Arcade Zone Season5 Round 5"},{"id":12,"round_event_nm":"Arcade Zone Season5 Round 6"}]};</script>
</body></html>`

func TestParseRoundBoard(t *testing.T) {
	cases := []struct {
		name    string
		html    string
		wantN   int // 赛季 5 的回合数，wantErr 时忽略
		wantErr bool
	}{
		{"正常内嵌", roundsFixtureHTML, 6, false},
		{"缺少数据", csrfHTMLNameFirst, 0, true},
		{"数据为空", `<script>window.roundsBySeason = {};</script>`, 0, true},
		{"数据损坏", `<script>window.roundsBySeason = {"5":not-json};</script>`, 0, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			board, err := parseRoundBoard(tt.html)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际 board=%v", board)
				}
				return
			}
			if err != nil {
				t.Fatalf("解析失败：%v", err)
			}
			if got := len(board["5"]); got != tt.wantN {
				t.Errorf("赛季 5 应有 %d 个回合，实际 %d", tt.wantN, got)
			}
		})
	}
}

func TestLookupRoundID(t *testing.T) {
	board, err := parseRoundBoard(roundsFixtureHTML)
	if err != nil {
		t.Fatalf("样例解析失败：%v", err)
	}

	cases := []struct {
		name     string
		season   int
		roundSeq int
		wantID   int
		wantOK   bool
	}{
		{"赛季5回合1", 5, 1, 5, true},
		{"赛季5回合4", 5, 4, 9, true},
		{"跳号回合（内置表覆盖不到）", 5, 5, 10, true},
		{"赛季5回合6", 5, 6, 12, true},
		{"赛季4回合2", 4, 2, 6, true},
		{"不存在的赛季", 9, 1, 0, false},
		{"赛季4无第3回合", 4, 3, 0, false},
		{"越界回合", 5, 8, 0, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lookupRoundID(board, tt.season, tt.roundSeq)
			if ok != tt.wantOK || (ok && got != tt.wantID) {
				t.Errorf("lookupRoundID(%d,%d) = (%d,%v)，期望 (%d,%v)",
					tt.season, tt.roundSeq, got, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestResolveRoundIDFallbackWithoutBoard(t *testing.T) {
	// 页面无内嵌映射时回退内置表（旧版行为）
	c := NewClient("Rin", 5)
	id, dynamic, err := c.resolveRoundID(5, 2)
	if err != nil {
		t.Fatalf("回退解析不应报错：%v", err)
	}
	if dynamic {
		t.Error("无页面数据时 dynamic 应为 false")
	}
	if id != 7 {
		t.Errorf("内置表 2→7，实际 %d", id)
	}
}
