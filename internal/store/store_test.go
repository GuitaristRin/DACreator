package store

import (
	"path/filepath"
	"testing"

	"github.com/GuitaristRin/DACreator/internal/model"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("打开失败：%v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func sample() []model.Record {
	return []model.Record{
		{Course: "秋名湖", Direction: "左周り", TimeMs: 147760, Rank: "EXPERT", Car: "CIVIC", National: "255", Date: "2026/01/19"},
		{Course: "秋名", Direction: "下坡", TimeMs: 152300, Rank: "LEGEND", Car: "AE86", National: "1", Date: "2025/12/21"},
	}
}

func TestInsertAndDedup(t *testing.T) {
	s := openTestStore(t)

	n, err := s.InsertRecords(sample(), "crawl")
	if err != nil || n != 2 {
		t.Fatalf("首次插入应 2 条：n=%d err=%v", n, err)
	}
	n, err = s.InsertRecords(sample(), "crawl")
	if err != nil || n != 0 {
		t.Fatalf("重复插入应全部去重：n=%d err=%v", n, err)
	}
	// 同赛道同方向不同成绩 → 新记录
	recs := sample()
	recs[0].TimeMs = 140000
	n, err = s.InsertRecords(recs, "crawl")
	if err != nil || n != 1 {
		t.Fatalf("不同成绩应插入 1 条：n=%d err=%v", n, err)
	}
}

func TestHistoryFilterAndLimit(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.InsertRecords(sample(), "search"); err != nil {
		t.Fatal(err)
	}

	all, err := s.History("", 100)
	if err != nil || len(all) != 2 {
		t.Fatalf("全部历史应 2 条：%d %v", len(all), err)
	}
	// 同秒插入时按 id 倒序：后插入的在前
	if all[0].TimeStr != "2:32.300" || all[1].TimeStr != "2:27.760" {
		t.Errorf("time_str 应为规范格式：%q / %q", all[0].TimeStr, all[1].TimeStr)
	}
	if all[0].Source != "search" {
		t.Errorf("source 不符：%q", all[0].Source)
	}

	onlyOne, err := s.History("秋名湖", 100)
	if err != nil || len(onlyOne) != 1 || onlyOne[0].Course != "秋名湖" {
		t.Fatalf("按赛道筛选不符：%+v %v", onlyOne, err)
	}

	limited, err := s.History("", 1)
	if err != nil || len(limited) != 1 {
		t.Fatalf("limit 不符：%d %v", len(limited), err)
	}
}

func TestDistinctCourses(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.InsertRecords(sample(), "crawl"); err != nil {
		t.Fatal(err)
	}
	courses, err := s.DistinctCourses()
	if err != nil {
		t.Fatal(err)
	}
	if len(courses) != 2 || courses[0] != "秋名" || courses[1] != "秋名湖" {
		t.Errorf("赛道列表不符：%v", courses)
	}
}
