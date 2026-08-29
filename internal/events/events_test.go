package events

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/GuitaristRin/DACreator/internal/az"
)

// 编译期保证 Emitter 满足爬虫的 Reporter 接口（协议收敛点）。
var _ az.Reporter = (*Emitter)(nil)

func TestJSONLines(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf, true)
	e.Progress("crawl", 42, "赛道 12/48")
	e.Log(LevelWarning, "赛道 6 抓取失败：超时")
	e.Error("network", "无法连接服务器")
	e.Result("out/raw/a.csv", "out/img/a.png", 12, 5230*time.Millisecond)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("应输出 4 行事件，实际 %d", len(lines))
	}

	var p Event
	if err := json.Unmarshal([]byte(lines[0]), &p); err != nil {
		t.Fatal(err)
	}
	if p.Type != TypeProgress || p.Stage != "crawl" || p.Pct != 42 || p.Detail != "赛道 12/48" {
		t.Errorf("progress 事件不符：%+v", p)
	}

	var lg Event
	_ = json.Unmarshal([]byte(lines[1]), &lg)
	if lg.Type != TypeLog || lg.Level != LevelWarning || lg.Msg != "赛道 6 抓取失败：超时" {
		t.Errorf("log 事件不符：%+v", lg)
	}

	var er Event
	_ = json.Unmarshal([]byte(lines[2]), &er)
	if er.Type != TypeError || er.Code != "network" {
		t.Errorf("error 事件不符：%+v", er)
	}

	var rs Event
	_ = json.Unmarshal([]byte(lines[3]), &rs)
	if rs.Type != TypeResult || rs.Records != 12 || rs.ElapsedMs != 5230 ||
		rs.CSVPath != "out/raw/a.csv" || rs.PNGPath != "out/img/a.png" {
		t.Errorf("result 事件不符：%+v", rs)
	}
}

func TestTextMode(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf, false)
	e.Progress("crawl", 42, "赛道 12/48")
	e.Log(LevelSuccess, "完成")
	e.Error("config", "未配置 ID")

	out := buf.String()
	if !strings.Contains(out, "[爬取 42%] 赛道 12/48") {
		t.Errorf("进度文本不符：\n%s", out)
	}
	if !strings.Contains(out, "✅ 完成") {
		t.Errorf("成功日志不符：\n%s", out)
	}
	if !strings.Contains(out, "❌ [config] 未配置 ID") {
		t.Errorf("错误文本不符：\n%s", out)
	}
	if strings.Contains(out, `"type"`) {
		t.Errorf("文本模式不应输出 JSON：\n%s", out)
	}
}
