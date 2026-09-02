// 回合 ID 的动态解析。
// 排行榜页面（与 CSRF 同一页面）内嵌 window.roundsBySeason —— 官方前端填充
// 回合下拉框所用的同源数据：{赛季: [{id: round_id, round_event_nm: "…Round N"}]}。
// 旧版把实测映射硬编码进程序，赛季更替即失效（且已漏掉 Season 5 的第 5-8 回合）；
// v3 起改为运行时解析，仅当页面不可达或结构变化时回退到内置 RoundIDMapping。
package az

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// RoundMeta 是页面内嵌的单条回合数据。
type RoundMeta struct {
	ID   int    `json:"id"`
	Name string `json:"round_event_nm"`
}

var (
	roundsBySeasonRe = regexp.MustCompile(`(?s)window\.roundsBySeason\s*=\s*(\{.*?\});`)
	roundNoRe        = regexp.MustCompile(`Round\s*(\d+)`)
)

// parseRoundBoard 从排行榜页面 HTML 提取内嵌的赛季→回合映射。
func parseRoundBoard(html string) (map[string][]RoundMeta, error) {
	m := roundsBySeasonRe.FindStringSubmatch(html)
	if m == nil {
		return nil, errors.New("页面未内嵌回合数据（结构可能已变化）")
	}
	var board map[string][]RoundMeta
	if err := json.Unmarshal([]byte(m[1]), &board); err != nil {
		return nil, fmt.Errorf("解析回合数据: %w", err)
	}
	if len(board) == 0 {
		return nil, errors.New("回合数据为空")
	}
	return board, nil
}

// cacheRoundBoard 把页面解析出的回合映射缓存到 Client（随 initCSRF 的页面抓取顺带完成）。
func (c *Client) cacheRoundBoard(html string) {
	if board, err := parseRoundBoard(html); err == nil {
		c.roundsMu.Lock()
		c.roundsBoard = board
		c.roundsMu.Unlock()
	}
}

// roundBoard 返回已缓存的回合映射；false 表示尚无数据（页面未抓取过或结构变化）。
func (c *Client) roundBoard() (map[string][]RoundMeta, bool) {
	c.roundsMu.Lock()
	defer c.roundsMu.Unlock()
	return c.roundsBoard, c.roundsBoard != nil
}

// resolveRoundID 把（赛季, 回合序号）解析为 API round_id。
// 页面映射可用时按名称编号精确匹配（官方调整赛季/回合后自动跟随）；
// 无页面数据时回退内置映射并返回 dynamic=false，由调用方记警告。
func (c *Client) resolveRoundID(season, roundSeq int) (id int, dynamic bool, err error) {
	board, ok := c.roundBoard()
	if !ok {
		return roundID(roundSeq), false, nil
	}
	if id, found := lookupRoundID(board, season, roundSeq); found {
		return id, true, nil
	}
	return 0, true, fmt.Errorf("赛季 %d 映射中不存在第 %d 回合（页面提供回合：%s）",
		season, roundSeq, roundNumbers(board[strconv.Itoa(season)]))
}

// lookupRoundID 在页面映射中按回合名称里的编号精确匹配；
// 不按列表顺序兜底——列表存在缺口时会取到错误数据，宁可让调用方显式报错。
func lookupRoundID(board map[string][]RoundMeta, season, roundSeq int) (int, bool) {
	rounds, ok := board[strconv.Itoa(season)]
	if !ok {
		return 0, false
	}
	for _, r := range rounds {
		if m := roundNoRe.FindStringSubmatch(r.Name); m != nil {
			if n, e := strconv.Atoi(m[1]); e == nil && n == roundSeq {
				return r.ID, true
			}
		}
	}
	return 0, false
}

func roundNumbers(rounds []RoundMeta) string {
	nums := make([]string, 0, len(rounds))
	for _, r := range rounds {
		if m := roundNoRe.FindStringSubmatch(r.Name); m != nil {
			nums = append(nums, m[1])
		}
	}
	if len(nums) == 0 {
		return "未知"
	}
	return strings.Join(nums, "/")
}
