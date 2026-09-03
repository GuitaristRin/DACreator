package az

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GuitaristRin/DACreator/internal/model"
	"golang.org/x/text/unicode/norm"
)

const (
	DefaultWebURL = "https://arcadezone.cn/ranking#timetrial"
	DefaultAPIURL = "https://arcadezone.cn/ranking/timetrial"
	defaultUA     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
)

// ErrNoRecords 表示爬取完成但没有任何成绩（配置用户名不存在或赛季无成绩）。
var ErrNoRecords = errors.New("未找到任何成绩记录")

// Reporter 由调用方实现，用于接收爬取进度与日志（GUI / CLI 各自适配）。
type Reporter interface {
	Progress(stage string, pct int, detail string)
	Log(level, msg string)
}

// Client 是 ArcadeZone 计时赛搜索 API 客户端。
type Client struct {
	WebURL   string
	APIURL   string
	RoundURL string
	PrideURL string
	TeamURL  string

	username string
	season   int
	http     *http.Client
	headers  map[string]string
	maxRetry int

	// 回合映射缓存：initCSRF 抓取排行榜页面时顺带解析内嵌的官方映射
	roundsMu    sync.Mutex
	roundsBoard map[string][]RoundMeta
}

// NewClient 创建客户端；username 需已做 NFKC 归一化（config.Load 已保证）。
func NewClient(username string, season int) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		WebURL:   DefaultWebURL,
		APIURL:   DefaultAPIURL,
		RoundURL: DefaultRoundURL,
		PrideURL: DefaultPrideURL,
		TeamURL:  DefaultTeamURL,
		username: username,
		season:   season,
		http:     &http.Client{Timeout: 30 * time.Second, Jar: jar},
		headers: map[string]string{
			"User-Agent":   defaultUA,
			"Content-Type": "application/json",
			"Accept":       "application/json, text/plain, */*",
			"Referer":      DefaultWebURL,
			"Origin":       "https://arcadezone.cn",
		},
		maxRetry: 3,
	}
}

var (
	metaTagRe   = regexp.MustCompile(`(?is)<meta[^>]*>`)
	attrNameRe  = regexp.MustCompile(`(?i)name\s*=\s*["']?([^"'\s>]+)`)
	attrValueRe = regexp.MustCompile(`(?i)(?:content|value)\s*=\s*["']([^"']*)["']`)
)

// initCSRF 访问排行榜页面提取 meta[name=csrf-token]，后续请求携带 X-CSRF-TOKEN。
func (c *Client) initCSRF(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.WebURL, nil)
	if err != nil {
		return fmt.Errorf("构造 CSRF 请求: %w", err)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("访问排行榜页面失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("访问排行榜页面失败：HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("读取排行榜页面: %w", err)
	}
	token, err := extractCSRFToken(string(body))
	if err != nil {
		return err
	}
	c.headers["X-CSRF-TOKEN"] = token
	// 同一页面内嵌官方回合映射，顺手缓存（失败不影响 CSRF 流程）
	c.cacheRoundBoard(string(body))
	return nil
}

func extractCSRFToken(html string) (string, error) {
	for _, tag := range metaTagRe.FindAllString(html, -1) {
		m := attrNameRe.FindStringSubmatch(tag)
		if m == nil || !strings.EqualFold(m[1], "csrf-token") {
			continue
		}
		if v := attrValueRe.FindStringSubmatch(tag); v != nil && v[1] != "" {
			return v[1], nil
		}
	}
	return "", errors.New("网页中未找到 CSRF Token（页面结构可能已变化）")
}

// searchItem 是搜索接口返回的单条榜单记录。
type searchItem struct {
	CourseID   int             `json:"course_id"`
	StyleCarID json.RawMessage `json:"style_car_id"`
	GoalTime   int             `json:"goal_time"`
	PlayDt     string          `json:"play_dt"`
	EvalID     int             `json:"eval_id"`
	Rank       json.RawMessage `json:"rank"`
	UserInfo   struct {
		Username string `json:"username"`
	} `json:"userinfo"`
}

type searchResp struct {
	List      []searchItem      `json:"list"`
	CarStyles map[string]string `json:"carStyles"`
}

// rawToString 把接口里可能为字符串或数字的字段统一转成字符串。
func rawToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	return ""
}

// SearchCourse 在指定赛道搜索当前用户的成绩，返回精确匹配的记录。
// 搜索接口为子串匹配，必须按 NFKC 归一化后精确过滤，避免混入无关玩家。
// 取消 ctx 可中断请求与重试等待。
func (c *Client) SearchCourse(ctx context.Context, courseID int) ([]model.Record, error) {
	if err := c.ensureCSRF(ctx); err != nil {
		return nil, err
	}
	payload := map[string]any{
		"page":   1, // 服务器单页即返回全部匹配，翻页只会重复
		"name":   c.username,
		"season": c.season,
		"course": courseID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("编码请求: %w", err)
	}

	var lastErr error
	for retry := 0; retry < c.maxRetry; retry++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if retry > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(retry) * time.Second):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("构造搜索请求: %w", err)
		}
		for k, v := range c.headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("搜索请求失败: %w", err)
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("读取搜索响应: %w", readErr)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("搜索请求失败：HTTP %d", resp.StatusCode)
			continue
		}
		var parsed searchResp
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("解析搜索响应: %w", err)
		}
		return c.filterRecords(parsed), nil
	}
	return nil, lastErr
}

// courseName / courseDirection 带兜底的查表。
func courseName(id int) string {
	if n, ok := CourseNameMap[id]; ok {
		return n
	}
	return "未知赛道"
}

func courseDirection(id int) string {
	if d, ok := CourseDirectionMap[id]; ok {
		return d
	}
	return "未知方向"
}

// filterRecords 从接口响应中筛出精确匹配目标用户名的记录。
func (c *Client) filterRecords(resp searchResp) []model.Record {
	target := norm.NFKC.String(c.username)
	carStyles := resp.CarStyles

	records := make([]model.Record, 0)
	for _, item := range resp.List {
		name := norm.NFKC.String(item.UserInfo.Username)
		if name != target {
			continue
		}
		carID := rawToString(item.StyleCarID)
		car, ok := carStyles[carID]
		if !ok || car == "" {
			car = "未知车型"
		}
		playDate, _, _ := strings.Cut(item.PlayDt, " ")
		records = append(records, model.Record{
			Course:    courseName(item.CourseID),
			Direction: courseDirection(item.CourseID),
			TimeMs:    item.GoalTime,
			Rank:      RankName(item.EvalID),
			Car:       car,
			National:  rawToString(item.Rank),
			Date:      playDate,
		})
	}
	return records
}

// ensureCSRF 保证已取得 CSRF Token（幂等）。
func (c *Client) ensureCSRF(ctx context.Context) error {
	if _, ok := c.headers["X-CSRF-TOKEN"]; ok {
		return nil
	}
	return c.initCSRF(ctx)
}
