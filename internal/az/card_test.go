package az

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// cardTestServer 模拟 round/pride/team 三个端点。
func cardTestServer(t *testing.T) (*httptest.Server, *Client) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ranking", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(csrfHTMLNameFirst))
	})

	// round：目标用户在第 2 页（第 3 位）
	mux.HandleFunc("POST /ranking/round", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Page    int `json:"page"`
			RoundID int `json:"round_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload.RoundID != 5 {
			t.Errorf("round_id 应为 5（回合 1），实际 %d", payload.RoundID)
		}
		if payload.Page == 1 {
			fmt.Fprint(w, `{"list":[`+roundItem("其他玩家", 100)+`],"pagination":{"page":1,"per_page":2,"last_page":2}}`)
			return
		}
		fmt.Fprint(w, `{"list":[`+roundItem("路人甲", 90)+`,`+roundItem("Rin", 80)+`],"pagination":{"page":2,"per_page":2,"last_page":2}}`)
	})

	// pride：第 1 页即命中
	mux.HandleFunc("POST /ranking/pride", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Season int `json:"season"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload.Season != 5 {
			t.Errorf("season 应为 5，实际 %d", payload.Season)
		}
		fmt.Fprint(w, `{"list":[{"pride_point":1234,"userinfo":{"username":"Rin"}}],"pagination":{"per_page":15,"last_page":1}}`)
	})

	// team：按车队名匹配
	mux.HandleFunc("POST /ranking/team", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"list":[{"team_point":567,"league_rank_id":4,"teaminfo":{"team_name":"Project D"}}],"pagination":{"per_page":15,"last_page":1}}`)
	})

	// timetrial：计时赛记录
	mux.HandleFunc("POST /ranking/timetrial", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"list":[{"course_id":0,"style_car_id":"12","goal_time":147760,"play_dt":"2026/01/19","eval_id":13,"rank":255,"userinfo":{"username":"Rin"}}],"carStyles":{"12":"CIVIC TYPE R"}}`)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewClient("Rin", 5)
	c.WebURL = srv.URL + "/ranking"
	c.APIURL = srv.URL + "/ranking/timetrial"
	c.RoundURL = srv.URL + "/ranking/round"
	c.PrideURL = srv.URL + "/ranking/pride"
	c.TeamURL = srv.URL + "/ranking/team"
	return srv, c
}

func roundItem(name string, point int) string {
	return fmt.Sprintf(`{"point":%d,"userinfo":{"username":"%s"}}`, point, name)
}

func TestRoundInfoAcrossPages(t *testing.T) {
	_, c := cardTestServer(t)
	point, rank, err := c.RoundInfo(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("查询失败：%v", err)
	}
	if point != 80 {
		t.Errorf("回合分数应为 80，实际 %d", point)
	}
	// 第 2 页第 2 位 → per_page=2 → 排名 = 2 + 2 = 4
	if rank != 4 {
		t.Errorf("排名应为 4，实际 %d", rank)
	}
}

func TestPrideInfo(t *testing.T) {
	_, c := cardTestServer(t)
	value, rank, err := c.PrideInfo(context.Background(), nil)
	if err != nil {
		t.Fatalf("查询失败：%v", err)
	}
	if value != 1234 || rank != 1 {
		t.Errorf("名声/排名不符：%d/%d", value, rank)
	}
}

func TestTeamInfo(t *testing.T) {
	_, c := cardTestServer(t)
	score, level, rank, err := c.TeamInfo(context.Background(), "Project D", 1, nil)
	if err != nil {
		t.Fatalf("查询失败：%v", err)
	}
	if score != 567 || level != "GOLD" || rank != 1 {
		t.Errorf("车队信息不符：%d/%s/%d", score, level, rank)
	}
	if _, _, _, err := c.TeamInfo(context.Background(), "不存在的车队", 1, nil); err == nil {
		t.Error("未命中车队应报错")
	}
}

func TestTeamInfoNFKCName(t *testing.T) {
	_, c := cardTestServer(t)
	// 配置里写全角字符也应命中（NFKC 归一化后与服务器车队名一致）
	_, _, _, err := c.TeamInfo(context.Background(), "Ｐｒｏｊｅｃｔ　Ｄ", 1, nil)
	if err != nil {
		t.Fatalf("全角车队名应命中：%v", err)
	}
}

func TestFetchPagedStopsAtPageCap(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ranking", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(csrfHTMLNameFirst))
	})
	var pages atomic.Int32
	mux.HandleFunc("POST /ranking/pride", func(w http.ResponseWriter, _ *http.Request) {
		pages.Add(1)
		// 永远未命中且 last_page 失真为极大值：验证防御性翻页上限兜底
		fmt.Fprint(w, `{"list":[{"pride_point":1,"userinfo":{"username":"别人"}}],"pagination":{"per_page":15,"last_page":99999}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewClient("Rin", 5)
	c.WebURL = srv.URL + "/ranking"
	c.PrideURL = srv.URL + "/ranking/pride"

	orig := maxBoardPages
	maxBoardPages = 3
	defer func() { maxBoardPages = orig }()

	_, _, err := c.PrideInfo(context.Background(), nil)
	if !errors.Is(err, errNotFound) {
		t.Fatalf("超过翻页上限应返回 errNotFound，实际：%v", err)
	}
	if n := pages.Load(); n != 3 {
		t.Errorf("应在第 %d 页停止，实际请求 %d 页", maxBoardPages, n)
	}
}

// dynamicRoundServer 的 CSRF 页面内嵌官方回合映射（与真实页面同构），
// 用于验证 round_id 的动态解析与越界拒绝。
func dynamicRoundServer(t *testing.T) (*httptest.Server, *Client, *atomic.Int32) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ranking", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(csrfHTMLNameFirst +
			`<script>window.roundsBySeason = {"9":[{"id":91,"round_event_nm":"Arcade Zone Season9 Round 1"},{"id":93,"round_event_nm":"Arcade Zone Season9 Round 2"}]};</script>`))
	})
	var gotRoundID atomic.Int32
	mux.HandleFunc("POST /ranking/round", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			RoundID int `json:"round_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotRoundID.Store(int32(payload.RoundID))
		fmt.Fprint(w, `{"list":[{"point":50,"userinfo":{"username":"Rin"}}],"pagination":{"per_page":15,"last_page":1}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewClient("Rin", 9)
	c.WebURL = srv.URL + "/ranking"
	c.RoundURL = srv.URL + "/ranking/round"
	return srv, c, &gotRoundID
}

func TestRoundInfoUsesDynamicRoundID(t *testing.T) {
	_, c, got := dynamicRoundServer(t)
	// 第 2 回合 → round_id=93：内置映射不可能给出的值
	point, rank, err := c.RoundInfo(context.Background(), 2, nil)
	if err != nil {
		t.Fatalf("查询失败：%v", err)
	}
	if point != 50 || rank != 1 {
		t.Errorf("回合数据不符：%d/%d", point, rank)
	}
	if id := got.Load(); id != 93 {
		t.Errorf("应使用页面映射的 round_id=93，实际 %d", id)
	}
}

func TestRoundInfoRejectsRoundBeyondBoard(t *testing.T) {
	_, c, _ := dynamicRoundServer(t)
	// 页面只定义第 1-2 回合：请求第 3 回合应显式报错，而不是静默回退到错误数据
	if _, _, err := c.RoundInfo(context.Background(), 3, nil); err == nil {
		t.Fatal("越界回合应报错")
	}
}

func TestFetchCardDataToleratesSecondaryFailures(t *testing.T) {
	_, c := cardTestServer(t)
	// 未配置车队名 → 车队信息失败但不应阻断
	data, err := c.FetchCardData(context.Background(), "", 1, nil)
	if err != nil {
		t.Fatalf("次要数据失败不应阻断：%v", err)
	}
	if len(data.Records) == 0 {
		t.Error("计时赛记录不应为空")
	}
	if data.PrideValue != 1234 {
		t.Errorf("名声值不符：%d", data.PrideValue)
	}
}
