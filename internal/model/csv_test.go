package model

import (
	"bytes"
	"strings"
	"testing"
)

func sampleRecords() []Record {
	return []Record{
		{Course: "秋名湖", Direction: "左周り", TimeMs: 147760, Rank: "EXPERT", Car: "CIVIC TYPE R (FL5) [HC]", National: "255位", Date: "2026/01/19"},
		{Course: "秋名湖", Direction: "右周り", TimeMs: 148702, Rank: "EXPERT", Car: "CIVIC TYPE R (FL5) [HC]", National: "121位", Date: "2025/12/21"},
	}
}

func TestCSVRoundTrip(t *testing.T) {
	in := sampleRecords()
	b, err := RecordToCSVBytes(in)
	if err != nil {
		t.Fatalf("写出 CSV 失败：%v", err)
	}
	if !bytes.HasPrefix(b, utf8BOM) {
		t.Fatalf("输出 CSV 应带 UTF-8 BOM")
	}
	got, err := ParseCSV(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("读回 CSV 失败：%v", err)
	}
	if len(got) != len(in) {
		t.Fatalf("记录数不符：got %d want %d", len(got), len(in))
	}
	for i := range in {
		if got[i] != in[i] {
			t.Errorf("第 %d 条不符：got %+v want %+v", i, got[i], in[i])
		}
	}
}

func TestParseCSVToleratesNoBOMAndCRLF(t *testing.T) {
	data := "コース,ルート,タイム,タイム評価,記録車種,全国順位,記録日\r\n秋名,下坡,2'27\"760,EXPERT,AE86,1位,2026/01/19\r\n"
	got, err := ParseCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if len(got) != 1 {
		t.Fatalf("应有 1 条记录，得到 %d", len(got))
	}
	r := got[0]
	if r.TimeMs != 147760 || r.National != "1位" || r.Course != "秋名" {
		t.Errorf("字段解析不符：%+v", r)
	}
}

func TestParseCSVAcceptsSixColumnSchema(t *testing.T) {
	// 旧版搜索模式输出无全国順位列
	data := "コース,ルート,タイム,タイム評価,記録車種,記録日\n秋名,下坡,1:02.345,MASTER,AE86,2026/01/19\n"
	got, err := ParseCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("解析 6 列 schema 失败：%v", err)
	}
	if got[0].National != "" || got[0].TimeMs != 62345 {
		t.Errorf("6 列解析结果不符：%+v", got[0])
	}
}

func TestParseCSVRejectsMissingColumn(t *testing.T) {
	data := "コース,ルート,タイム,タイム評価,記録日\n秋名,下坡,1:02.345,MASTER,2026/01/19\n"
	_, err := ParseCSV(strings.NewReader(data))
	if err == nil || !strings.Contains(err.Error(), "缺少必要列") {
		t.Fatalf("应报缺少必要列错误，实际：%v", err)
	}
}

func TestParseCSVReportsBadTimeWithLineNumber(t *testing.T) {
	data := "コース,ルート,タイム,タイム評価,記録車種,全国順位,記録日\n秋名,下坡,垃圾,MASTER,AE86,1位,2026/01/19\n"
	_, err := ParseCSV(strings.NewReader(data))
	if err == nil || !strings.Contains(err.Error(), "第 2 行") {
		t.Fatalf("错误信息应带行号，实际：%v", err)
	}
}

func TestParseCSVSkipsBlankLines(t *testing.T) {
	data := "コース,ルート,タイム,タイム評価,記録車種,全国順位,記録日\n\n秋名,下坡,1:02.345,MASTER,AE86,1位,2026/01/19\n\n"
	got, err := ParseCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if len(got) != 1 {
		t.Fatalf("应跳过空行仅剩 1 条，得到 %d", len(got))
	}
}
