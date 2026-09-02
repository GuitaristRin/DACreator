package az

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/GuitaristRin/DACreator/internal/model"
	"golang.org/x/text/unicode/norm"
)

// 成绩卡相关端点默认值（与旧版 round/pride/team 模块的接口一致）。
const (
	DefaultRoundURL = "https://arcadezone.cn/ranking/round"
	DefaultPrideURL = "https://arcadezone.cn/ranking/pride"
	DefaultTeamURL  = "https://arcadezone.cn/ranking/team"
)

// RoundIDMapping 回合序号(1-4) → API round_id（旧版实测映射，官方调整赛季时需维护）。
var RoundIDMapping = map[int]int{1: 5, 2: 7, 3: 8, 4: 9}

// LeagueMapping 联赛等级 ID → 名称（来源 team_league0-6.png 编号）。
var LeagueMapping = map[int]string{
	0: "OPEN", 1: "BASIC", 2: "BRONZE", 3: "SILVER",
	4: "GOLD", 5: "PLATINUM", 6: "MASTER",
}

func roundID(seq int) int {
	if id, ok := RoundIDMapping[seq]; ok {
		return id
	}
	return 9 // 旧版默认回合 4 → 9
}

// postJSON 带重试的 POST JSON，返回解析后的响应对象。
// 取消 ctx 可中断请求与重试等待。
func (c *Client) postJSON(ctx context.Context, url string, payload map[string]any) (map[string]any, error) {
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
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		for k, v := range c.headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("解析响应: %w", err)
		}
		return parsed, nil
	}
	return nil, lastErr
}

// maxBoardPages 是 fetchPaged 的防御性翻页上限：正常由服务器 pagination.last_page
// 终止翻页，此上限仅在端点异常（last_page 失真）时兜底，避免无限请求。
// 可在测试中临时调小。
var maxBoardPages = 500

// fetchPaged 逐页请求端点，直到 match 命中或翻完 last_page。
// 返回命中的 item 与其全国排名（按页内位置计算）。
func (c *Client) fetchPaged(ctx context.Context, url string, makePayload func(page int) map[string]any, match func(item map[string]any) bool) (map[string]any, int, error) {
	for page := 1; page <= maxBoardPages; page++ {
		data, err := c.postJSON(ctx, url, makePayload(page))
		if err != nil {
			return nil, 0, err
		}
		list, _ := data["list"].([]any)
		pagination := subMap(data, "pagination")
		perPage := int(toFloat(pagination, "per_page", 15))
		lastPage := int(toFloat(pagination, "last_page", 1))

		for idx, raw := range list {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if match(item) {
				return item, (page-1)*perPage + idx + 1, nil
			}
		}
		if page >= lastPage {
			return nil, 0, errNotFound
		}
	}
	return nil, 0, errNotFound
}

var errNotFound = fmt.Errorf("未找到匹配记录")

func toFloat(m map[string]any, key string, def float64) float64 {
	if m == nil {
		return def
	}
	if v, ok := m[key].(float64); ok {
		return v
	}
	return def
}

func subMap(m map[string]any, key string) map[string]any {
	if sub, ok := m[key].(map[string]any); ok {
		return sub
	}
	return map[string]any{}
}

func (c *Client) usernameMatches(rawName any) bool {
	name, _ := rawName.(string)
	return norm.NFKC.String(name) == norm.NFKC.String(c.username)
}

// RoundInfo 查询用户在指定回合（1-4）的分数与排名。
func (c *Client) RoundInfo(ctx context.Context, roundSeq int, rep Reporter) (point, rank int, err error) {
	if err := c.ensureCSRF(ctx); err != nil {
		return 0, 0, err
	}
	if rep == nil {
		rep = nopReporter{}
	}
	id := roundID(roundSeq)
	rep.Log("info", fmt.Sprintf("查询第 %d 回合排名（round_id=%d）", roundSeq, id))

	item, rankPos, err := c.fetchPaged(ctx, c.RoundURL,
		func(page int) map[string]any { return map[string]any{"page": page, "round_id": id} },
		func(item map[string]any) bool { return c.usernameMatches(subMap(item, "userinfo")["username"]) },
	)
	if err != nil {
		return 0, 0, fmt.Errorf("第 %d 回合：%w", roundSeq, err)
	}
	return int(toFloat(item, "point", 0)), rankPos, nil
}

// PrideInfo 查询用户在当前赛季的名声值与排名。
func (c *Client) PrideInfo(ctx context.Context, rep Reporter) (value, rank int, err error) {
	if err := c.ensureCSRF(ctx); err != nil {
		return 0, 0, err
	}
	if rep == nil {
		rep = nopReporter{}
	}

	item, rankPos, err := c.fetchPaged(ctx, c.PrideURL,
		func(page int) map[string]any { return map[string]any{"page": page, "season": c.season} },
		func(item map[string]any) bool { return c.usernameMatches(subMap(item, "userinfo")["username"]) },
	)
	if err != nil {
		return 0, 0, fmt.Errorf("名声排名：%w", err)
	}
	return int(toFloat(item, "pride_point", 0)), rankPos, nil
}

// TeamInfo 按车队名查询车队的分数、联赛等级与排名。
func (c *Client) TeamInfo(ctx context.Context, teamName string, roundSeq int, rep Reporter) (score int, level string, rank int, err error) {
	if err := c.ensureCSRF(ctx); err != nil {
		return 0, "", 0, err
	}
	if rep == nil {
		rep = nopReporter{}
	}
	if teamName == "" {
		return 0, "", 0, fmt.Errorf("未配置车队名称")
	}
	id := roundID(roundSeq)
	rep.Log("info", fmt.Sprintf("查询车队 %s 排名（round_id=%d）", teamName, id))

	// 与用户名一致做 NFKC 归一化精确匹配，容忍全角/半角差异
	target := norm.NFKC.String(teamName)
	item, rankPos, err := c.fetchPaged(ctx, c.TeamURL,
		func(page int) map[string]any { return map[string]any{"page": page, "round_id": id} },
		func(item map[string]any) bool {
			name, _ := subMap(item, "teaminfo")["team_name"].(string)
			return norm.NFKC.String(name) == target
		},
	)
	if err != nil {
		return 0, "", 0, fmt.Errorf("车队排名：%w", err)
	}
	leagueID := int(toFloat(item, "league_rank_id", 0))
	level, ok := LeagueMapping[leagueID]
	if !ok {
		level = fmt.Sprintf("LEVEL_%d", leagueID)
	}
	return int(toFloat(item, "team_point", 0)), level, rankPos, nil
}

// CardData 是成绩卡需要的全部数据。
type CardData struct {
	Records    []model.Record
	RoundSeq   int
	RoundScore int
	RoundRank  int
	PrideValue int
	PrideRank  int
	TeamName   string
	TeamScore  int
	TeamLevel  string
	TeamRank   int
}

// FetchCardData 抓取成绩卡所需数据：计时赛记录（必需）+ 回合/名声/车队（尽力而为）。
// 次要数据失败只记警告，不阻断出卡（与旧版行为一致）。
func (c *Client) FetchCardData(ctx context.Context, cfgTeamName string, cfgRound int, rep Reporter) (CardData, error) {
	if rep == nil {
		rep = nopReporter{}
	}
	data := CardData{RoundSeq: cfgRound, TeamName: cfgTeamName}

	records, err := c.CrawlAll(ctx, rep, DefaultConcurrency)
	if err != nil {
		return data, err
	}
	data.Records = records

	if pt, rk, err := c.RoundInfo(ctx, cfgRound, rep); err != nil {
		rep.Log("warning", "回合信息获取失败："+err.Error())
	} else {
		data.RoundScore, data.RoundRank = pt, rk
	}
	if v, rk, err := c.PrideInfo(ctx, rep); err != nil {
		rep.Log("warning", "名声信息获取失败："+err.Error())
	} else {
		data.PrideValue, data.PrideRank = v, rk
	}
	if s, lv, rk, err := c.TeamInfo(ctx, cfgTeamName, cfgRound, rep); err != nil {
		rep.Log("warning", "车队信息获取失败："+err.Error())
	} else {
		data.TeamScore, data.TeamLevel, data.TeamRank = s, lv, rk
	}
	return data, nil
}
