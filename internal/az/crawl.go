package az

import (
	"context"
	"fmt"
	"sync"

	"github.com/GuitaristRin/DACreator/internal/model"
)

// DefaultConcurrency 是并发抓取赛道的默认协程数。
// 目标站点是社区服务器，保持克制，避免给对方造成压力。
const DefaultConcurrency = 6

// CrawlAll 并发抓取全部赛道上目标用户的成绩，结果按赛道 ID 升序排列。
// 单个赛道失败只记警告并继续；全部赛道都无记录时返回 ErrNoRecords。
func (c *Client) CrawlAll(ctx context.Context, rep Reporter, concurrency int) ([]model.Record, error) {
	if err := c.ensureCSRF(); err != nil {
		return nil, err
	}
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	if rep == nil {
		rep = nopReporter{}
	}

	type courseResult struct {
		idx     int
		records []model.Record
		err     error
	}

	total := len(TargetCourses)
	results := make([]courseResult, total)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0

	for i, courseID := range TargetCourses {
		wg.Add(1)
		go func(idx, cid int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			records, err := c.SearchCourse(cid)
			mu.Lock()
			results[idx] = courseResult{idx: idx, records: records, err: err}
			done++
			pct := done * 100 / total
			mu.Unlock()
			if err != nil {
				rep.Log("warning", fmt.Sprintf("赛道 %s（ID %d）抓取失败：%v", courseName(cid), cid, err))
				rep.Progress("crawl", pct, fmt.Sprintf("赛道 %d/%d", done, total))
				return
			}
			if len(records) > 0 {
				rep.Log("info", fmt.Sprintf("%s（%s）找到 %d 条记录", courseName(cid), courseDirection(cid), len(records)))
			}
			rep.Progress("crawl", pct, fmt.Sprintf("赛道 %d/%d", done, total))
		}(i, courseID)
	}
	wg.Wait()

	all := make([]model.Record, 0, 64)
	for _, r := range results {
		all = append(all, r.records...)
	}
	if len(all) == 0 {
		return nil, ErrNoRecords
	}
	rep.Log("info", fmt.Sprintf("爬取完成，共 %d 条成绩记录", len(all)))
	return all, nil
}

type nopReporter struct{}

func (nopReporter) Progress(stage string, pct int, detail string) {}
func (nopReporter) Log(level, msg string)                         {}
