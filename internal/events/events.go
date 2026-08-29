// Package events 定义引擎对 GUI / --json 消费者的事件契约（JSON-lines，每行一个事件）。
// 该包是 GUI↔引擎的唯一协议，字段变更必须同步 AGENT.md 的示例。
package events

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// 事件类型。
const (
	TypeProgress = "progress"
	TypeLog      = "log"
	TypeResult   = "result"
	TypeError    = "error"
)

// 日志级别。
const (
	LevelInfo    = "info"
	LevelWarning = "warning"
	LevelError   = "error"
	LevelSuccess = "success"
)

// Event 是单条事件。
type Event struct {
	Type   string `json:"type"`
	Stage  string `json:"stage,omitempty"`  // progress：crawl / render / save / card
	Pct    int    `json:"pct,omitempty"`    // progress：0-100
	Detail string `json:"detail,omitempty"` // progress：人读细节
	Level  string `json:"level,omitempty"`  // log：级别
	Msg    string `json:"msg,omitempty"`    // log / error：消息
	Code   string `json:"code,omitempty"`   // error：network / config / io / parse
	CSVPath string `json:"csv_path,omitempty"` // result
	PNGPath string `json:"png_path,omitempty"` // result
	Records int    `json:"records,omitempty"`  // result
	ElapsedMs int64 `json:"elapsed_ms,omitempty"` // result
}

// Emitter 把事件写到输出流：JSON 模式逐行输出事件，文本模式输出人读日志。
// Emitter 实现 az.Reporter。
type Emitter struct {
	mu       sync.Mutex
	w        io.Writer
	jsonMode bool
}

// NewEmitter 创建事件输出器。jsonMode=true 时输出 JSON-lines，否则输出人读文本。
func NewEmitter(w io.Writer, jsonMode bool) *Emitter {
	return &Emitter{w: w, jsonMode: jsonMode}
}

// Progress 输出进度事件。
func (e *Emitter) Progress(stage string, pct int, detail string) {
	e.emit(Event{Type: TypeProgress, Stage: stage, Pct: pct, Detail: detail})
}

// Log 输出日志事件。
func (e *Emitter) Log(level, msg string) {
	e.emit(Event{Type: TypeLog, Level: level, Msg: msg})
}

// Result 输出任务完成事件。
func (e *Emitter) Result(csvPath, pngPath string, records int, elapsed time.Duration) {
	e.emit(Event{
		Type: TypeResult, CSVPath: csvPath, PNGPath: pngPath,
		Records: records, ElapsedMs: elapsed.Milliseconds(),
	})
}

// Error 输出错误事件。
func (e *Emitter) Error(code, msg string) {
	e.emit(Event{Type: TypeError, Code: code, Msg: msg})
}

func (e *Emitter) emit(ev Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.jsonMode {
		data, err := json.Marshal(ev)
		if err != nil {
			return
		}
		fmt.Fprintf(e.w, "%s\n", data)
		return
	}
	fmt.Fprintln(e.w, humanize(ev))
}

// humanize 把事件转成人读文本（对齐旧版 CLI 的 emoji 风格）。
func humanize(ev Event) string {
	switch ev.Type {
	case TypeProgress:
		stage := stageName(ev.Stage)
		if ev.Detail != "" {
			return fmt.Sprintf("[%s %d%%] %s", stage, ev.Pct, ev.Detail)
		}
		return fmt.Sprintf("[%s %d%%]", stage, ev.Pct)
	case TypeLog:
		return fmt.Sprintf("%s %s", levelEmoji(ev.Level), ev.Msg)
	case TypeError:
		return fmt.Sprintf("❌ [%s] %s", ev.Code, ev.Msg)
	case TypeResult:
		out := "✅ 任务完成"
		if ev.Records > 0 {
			out += fmt.Sprintf("，共 %d 条记录", ev.Records)
		}
		if ev.CSVPath != "" {
			out += fmt.Sprintf("\n   CSV：%s", ev.CSVPath)
		}
		if ev.PNGPath != "" {
			out += fmt.Sprintf("\n   图片：%s", ev.PNGPath)
		}
		out += fmt.Sprintf("\n   耗时：%s", (time.Duration(ev.ElapsedMs) * time.Millisecond).String())
		return out
	}
	return fmt.Sprintf("%v", ev)
}

func stageName(stage string) string {
	switch stage {
	case "crawl":
		return "爬取"
	case "render":
		return "渲染"
	case "save":
		return "保存"
	case "card":
		return "成绩卡"
	}
	return stage
}

func levelEmoji(level string) string {
	switch level {
	case LevelSuccess:
		return "✅"
	case LevelWarning:
		return "⚠️"
	case LevelError:
		return "❌"
	}
	return "📌"
}
