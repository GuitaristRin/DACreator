package az

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

const csrfHTMLNameFirst = `<!doctype html><html><head><meta charset="utf-8"><meta name="csrf-token" content="tok-abc123">` +
	`<meta name="viewport" content="width=device-width"></head><body></body></html>`

const csrfHTMLContentFirst = `<!doctype html><html><head><meta content="tok-xyz789" name="csrf-token"></head><body></body></html>`

func TestExtractCSRFToken(t *testing.T) {
	cases := []struct {
		name    string
		html    string
		want    string
		wantErr bool
	}{
		{"name 在前", csrfHTMLNameFirst, "tok-abc123", false},
		{"content 在前", csrfHTMLContentFirst, "tok-xyz789", false},
		{"缺失", `<html><head><meta name="other" content="x"></head></html>`, "", true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractCSRFToken(tt.html)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际得到 %q", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("got %q err %v, want %q", got, err, tt.want)
			}
		})
	}
}

// newTestServer 起一个同时提供 CSRF 页面与搜索 API 的测试服务器。
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ranking", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(csrfHTMLNameFirst))
	})
	mux.HandleFunc("POST /ranking/timetrial", handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewClient("Rin", 5)
	c.WebURL = srv.URL + "/ranking"
	c.APIURL = srv.URL + "/ranking/timetrial"
	return srv, c
}

const searchFixture = `{
  "list": [
    {"course_id": 0, "style_car_id": "12", "goal_time": 147760, "play_dt": "2026/01/19 12:00:00", "eval_id": 13, "rank": 255, "userinfo": {"username": "Rin"}},
    {"course_id": 0, "style_car_id": 34, "goal_time": 148000, "play_dt": "2025/12/21 10:00:00", "eval_id": 29, "rank": "121", "userinfo": {"username": "Rin"}},
    {"course_id": 0, "style_car_id": "99", "goal_time": 100000, "play_dt": "2025/12/01 10:00:00", "eval_id": 1, "rank": 1, "userinfo": {"username": "ななせや"}}
  ],
  "carStyles": {"12": "CIVIC TYPE R (FL5) [HC]", "34": "AE86"}
}`

func TestSearchCourseFiltersAndMaps(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-TOKEN") != "tok-abc123" {
			t.Errorf("请求应携带 CSRF Token，实际 %q", r.Header.Get("X-CSRF-TOKEN"))
		}
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") == false {
			t.Errorf("Content-Type 应为 JSON：%q", r.Header.Get("Content-Type"))
		}
		w.Write([]byte(searchFixture))
	})

	records, err := c.SearchCourse(0)
	if err != nil {
		t.Fatalf("搜索失败：%v", err)
	}
	if len(records) != 2 {
		t.Fatalf("应精确过滤出 2 条记录（含重名保留），实际 %d", len(records))
	}
	r0, r1 := records[0], records[1]
	if r0.Course != "秋名湖" || r0.Direction != "逆时针" {
		t.Errorf("赛道映射不符：%+v", r0)
	}
	if r0.TimeMs != 147760 || r0.Rank != "EXPERT" || r0.National != "255" {
		t.Errorf("第 1 条字段不符：%+v", r0)
	}
	if r0.Date != "2026/01/19" {
		t.Errorf("日期应去掉时间部分：%q", r0.Date)
	}
	if r1.Rank != "LEGEND" || r1.Car != "AE86" || r1.National != "121" {
		t.Errorf("第 2 条字段不符（数字车型 ID 与字符串排名）：%+v", r1)
	}
}

func TestSearchCourseUnknownCarFallsBack(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"list":[{"course_id":999,"style_car_id":"404","goal_time":1,"play_dt":"2026/01/01","eval_id":99,"rank":null,"userinfo":{"username":"Rin"}}],"carStyles":{}}`))
	})
	records, err := c.SearchCourse(0)
	if err != nil {
		t.Fatalf("搜索失败：%v", err)
	}
	r := records[0]
	if r.Course != "未知赛道" || r.Direction != "未知方向" || r.Car != "未知车型" || r.Rank != "未知评价" {
		t.Errorf("兜底映射不符：%+v", r)
	}
}

func TestSearchCourseRetriesOnServerError(t *testing.T) {
	var calls atomic.Int32
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(searchFixture))
	})
	c.maxRetry = 3
	// 重试间隔在测试里会真实 sleep（1s+2s），可接受；若嫌慢可缩短
	records, err := c.SearchCourse(0)
	if err != nil {
		t.Fatalf("重试后应成功：%v", err)
	}
	if len(records) != 2 {
		t.Errorf("应有 2 条记录，实际 %d", len(records))
	}
	if calls.Load() != 3 {
		t.Errorf("应请求 3 次，实际 %d", calls.Load())
	}
}

func TestCrawlAllOrderAndNoRecords(t *testing.T) {
	orig := TargetCourses
	TargetCourses = []int{2, 0}
	defer func() { TargetCourses = orig }()

	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Course int `json:"course"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		resp := `{"list":[{"course_id":` + strconv.Itoa(payload.Course) + `,"style_car_id":"1","goal_time":100000,"play_dt":"2026/01/01","eval_id":5,"rank":10,"userinfo":{"username":"Rin"}}],"carStyles":{"1":"AE86"}}`
		w.Write([]byte(resp))
	})

	records, err := c.CrawlAll(context.Background(), nil, 2)
	if err != nil {
		t.Fatalf("爬取失败：%v", err)
	}
	if len(records) != 2 {
		t.Fatalf("应有 2 条记录，实际 %d", len(records))
	}
	// 结果必须按赛道 ID 升序（0 在前，2 在后）
	if records[0].Course != "秋名湖" && records[1].Course != "秋名湖" {
		t.Fatalf("赛道名映射不符：%+v", records)
	}
	if records[0].National != "10" {
		t.Errorf("排名字段不符：%+v", records[0])
	}
}

func TestCrawlAllNoRecords(t *testing.T) {
	orig := TargetCourses
	TargetCourses = []int{0}
	defer func() { TargetCourses = orig }()

	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"list":[],"carStyles":{}}`))
	})
	_, err := c.CrawlAll(context.Background(), nil, 1)
	if !errors.Is(err, ErrNoRecords) {
		t.Fatalf("应返回 ErrNoRecords，实际：%v", err)
	}
}
