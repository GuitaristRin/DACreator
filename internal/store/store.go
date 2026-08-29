// Package store 管理历史成绩的 SQLite 存储（modernc.org/sqlite，纯 Go 无 cgo）。
// 表结构与旧版 extras/database.py 保持一致，旧版 dacreator_history.db 可直接复用。
package store

import (
	"database/sql"
	"fmt"

	"github.com/GuitaristRin/DACreator/internal/model"

	_ "modernc.org/sqlite"
)

// Store 封装历史数据库连接。
type Store struct {
	db *sql.DB
}

// HistoryRecord 是一条历史查询结果。
type HistoryRecord struct {
	ID           int64  `json:"id"`
	Course       string `json:"course"`
	Direction    string `json:"direction"`
	TimeStr      string `json:"time_str"`
	TimeMs       int    `json:"time_ms"`
	Rank         string `json:"rank"`
	Car          string `json:"car"`
	NationalRank string `json:"national_rank"`
	RecordDate   string `json:"record_date"`
	CreatedAt    string `json:"created_at"`
	Source       string `json:"source"`
}

// Open 打开（必要时创建）数据库并初始化表结构。
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("打开历史数据库 %s: %w", path, err)
	}
	s := &Store{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	const schema = `
CREATE TABLE IF NOT EXISTS records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	course TEXT NOT NULL,
	direction TEXT NOT NULL,
	time_str TEXT NOT NULL,
	time_ms INTEGER NOT NULL,
	rank TEXT,
	car TEXT,
	national_rank TEXT,
	record_date TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	source TEXT
);
CREATE INDEX IF NOT EXISTS idx_course_direction ON records(course, direction);
CREATE INDEX IF NOT EXISTS idx_time_ms ON records(time_ms);`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("初始化历史数据库: %w", err)
	}
	return nil
}

// Close 关闭连接。
func (s *Store) Close() error { return s.db.Close() }

// InsertRecords 批量写入记录；以 (course, direction, time_ms) 去重，
// 返回实际新插入的条数（与旧版"智能去重"行为一致）。
func (s *Store) InsertRecords(records []model.Record, source string) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启事务: %w", err)
	}
	defer tx.Rollback()

	var existsStmt, insertStmt *sql.Stmt
	existsStmt, err = tx.Prepare(`SELECT id FROM records WHERE course = ? AND direction = ? AND time_ms = ?`)
	if err != nil {
		return 0, fmt.Errorf("准备查询语句: %w", err)
	}
	defer existsStmt.Close()
	insertStmt, err = tx.Prepare(`INSERT INTO records
		(course, direction, time_str, time_ms, rank, car, national_rank, record_date, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("准备插入语句: %w", err)
	}
	defer insertStmt.Close()

	inserted := 0
	for _, r := range records {
		var id int64
		err := existsStmt.QueryRow(r.Course, r.Direction, r.TimeMs).Scan(&id)
		if err == nil {
			continue // 已存在，跳过
		}
		if err != sql.ErrNoRows {
			return inserted, fmt.Errorf("查询重复记录: %w", err)
		}
		if _, err := insertStmt.Exec(r.Course, r.Direction, model.FormatRaceTime(r.TimeMs),
			r.TimeMs, r.Rank, r.Car, r.National, r.Date, source); err != nil {
			return inserted, fmt.Errorf("插入记录: %w", err)
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		return inserted, fmt.Errorf("提交事务: %w", err)
	}
	return inserted, nil
}

// History 按赛道筛选（course 为空表示全部），按录入时间倒序，最多 limit 条。
func (s *Store) History(course string, limit int) ([]HistoryRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, course, direction, time_str, time_ms, rank, car, national_rank, record_date, created_at, source FROM records`
	args := []any{}
	if course != "" {
		query += ` WHERE course = ?`
		args = append(args, course)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询历史记录: %w", err)
	}
	defer rows.Close()

	out := []HistoryRecord{}
	for rows.Next() {
		var r HistoryRecord
		if err := rows.Scan(&r.ID, &r.Course, &r.Direction, &r.TimeStr, &r.TimeMs,
			&r.Rank, &r.Car, &r.NationalRank, &r.RecordDate, &r.CreatedAt, &r.Source); err != nil {
			return nil, fmt.Errorf("读取历史记录: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DistinctCourses 返回库中出现过的一切赛道名。
func (s *Store) DistinctCourses() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT course FROM records ORDER BY course`)
	if err != nil {
		return nil, fmt.Errorf("查询赛道列表: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
