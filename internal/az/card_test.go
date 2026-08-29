package az

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	point, rank, err := c.RoundInfo(1, nil)
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
	value, rank, err := c.PrideInfo(nil)
	if err != nil {
		t.Fatalf("查询失败：%v", err)
	}
	if value != 1234 || rank != 1 {
		t.Errorf("名声/排名不符：%d/%d", value, rank)
	}
}

func TestTeamInfo(t *testing.T) {
	_, c := cardTestServer(t)
	score, level, rank, err := c.TeamInfo("Project D", 1, nil)
	if err != nil {
		t.Fatalf("查询失败：%v", err)
	}
	if score != 567 || level != "GOLD" || rank != 1 {
		t.Errorf("车队信息不符：%d/%s/%d", score, level, rank)
	}
	if _, _, _, err := c.TeamInfo("不存在的车队", 1, nil); err == nil {
		t.Error("未命中车队应报错")
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
