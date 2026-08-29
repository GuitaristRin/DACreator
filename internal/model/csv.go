package model

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// utf8BOM 与旧版保持一致：写出带 BOM（Excel 兼容），读取容忍有无 BOM。
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// WriteCSV 将记录写出为 UTF-8（BOM）CSV，schema 与旧版一致。
// 时间列以规范格式 m:ss.mmm 写出。
func WriteCSV(w io.Writer, records []Record) error {
	if _, err := w.Write(utf8BOM); err != nil {
		return fmt.Errorf("写入 BOM: %w", err)
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(Columns); err != nil {
		return fmt.Errorf("写入表头: %w", err)
	}
	for _, r := range records {
		row := []string{
			r.Course,
			r.Direction,
			FormatRaceTime(r.TimeMs),
			r.Rank,
			r.Car,
			r.National,
			r.Date,
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("写入记录行: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// RecordToCSVBytes 是 WriteCSV 的便捷封装，用于事件流与 CLI 输出。
func RecordToCSVBytes(records []Record) ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, records); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ParseCSV 从 r 读取成绩 CSV。
// 兼容旧版两种 schema：7 列（含全国順位）与 6 列（缺全国順位，此时 National 为空）。
func ParseCSV(r io.Reader) ([]Record, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取 CSV 数据: %w", err)
	}
	data = bytes.TrimPrefix(data, utf8BOM)

	cr := csv.NewReader(bytes.NewReader(data))
	cr.FieldsPerRecord = -1 // 6/7 列并存，逐行校验
	cr.LazyQuotes = true    // 成绩字段含裸引号（如 2'27"760），与旧版 Python csv 行为对齐
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析 CSV: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("CSV 为空")
	}

	header := rows[0]
	colIdx := make(map[string]int, len(header))
	for i, name := range header {
		colIdx[strings.TrimSpace(name)] = i
	}
	// 必需列：除全国順位外的 6 列
	required := []string{ColCourse, ColDirection, ColTime, ColRank, ColCar, ColDate}
	for _, col := range required {
		if _, ok := colIdx[col]; !ok {
			return nil, fmt.Errorf("CSV 缺少必要列 %s（需包含：%s）", col, strings.Join(Columns, ","))
		}
	}
	nationalIdx, hasNational := colIdx[ColNational]

	get := func(row []string, i int) string {
		if i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	records := make([]Record, 0, len(rows)-1)
	for lineNo, row := range rows[1:] {
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue // 跳过空行
		}
		timeMs, err := ParseRaceTime(get(row, colIdx[ColTime]))
		if err != nil {
			return nil, fmt.Errorf("第 %d 行：%w", lineNo+2, err)
		}
		rec := Record{
			Course:    get(row, colIdx[ColCourse]),
			Direction: get(row, colIdx[ColDirection]),
			TimeMs:    timeMs,
			Rank:      get(row, colIdx[ColRank]),
			Car:       get(row, colIdx[ColCar]),
			Date:      get(row, colIdx[ColDate]),
		}
		if hasNational {
			rec.National = get(row, nationalIdx)
		}
		records = append(records, rec)
	}
	return records, nil
}
