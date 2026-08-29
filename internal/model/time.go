package model

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatRaceTime 将毫秒格式化为引擎规范格式 m:ss.mmm（与 ArcadeZone 服务器返回一致）。
func FormatRaceTime(ms int) string {
	m := ms / 60000
	s := (ms % 60000) / 1000
	milli := ms % 1000
	return fmt.Sprintf("%d:%02d.%03d", m, s, milli)
}

// ParseRaceTime 解析成绩时间字符串，兼容两种历史格式：
//   - m:ss.mmm（服务器格式，如 2:27.760）
//   - m'ss"mmm（旧版用户 CSV / rank.csv 格式，如 2'27"760）
func ParseRaceTime(s string) (int, error) {
	s = strings.TrimSpace(s)

	var minute, second, milli string
	switch {
	case strings.Contains(s, ":"):
		parts := strings.SplitN(s, ":", 2)
		minute = parts[0]
		rest := strings.SplitN(parts[1], ".", 2)
		if len(rest) != 2 {
			return 0, fmt.Errorf("无法解析时间 %q：缺少毫秒部分", s)
		}
		second, milli = rest[0], rest[1]
	case strings.Contains(s, "'") && strings.Contains(s, "\""):
		parts := strings.SplitN(s, "'", 2)
		minute = parts[0]
		rest := strings.SplitN(parts[1], "\"", 2)
		if len(rest) != 2 {
			return 0, fmt.Errorf("无法解析时间 %q：缺少毫秒部分", s)
		}
		second, milli = rest[0], rest[1]
	default:
		return 0, fmt.Errorf("无法解析时间 %q：格式应为 m:ss.mmm 或 m'ss\"mmm", s)
	}

	mm, err := strconv.Atoi(strings.TrimSpace(minute))
	if err != nil {
		return 0, fmt.Errorf("无法解析时间 %q：分钟部分无效", s)
	}
	ss, err := strconv.Atoi(strings.TrimSpace(second))
	if err != nil || ss < 0 || ss > 59 {
		return 0, fmt.Errorf("无法解析时间 %q：秒部分无效", s)
	}
	msPart, err := strconv.Atoi(strings.TrimSpace(milli))
	if err != nil || msPart < 0 || msPart > 999 {
		return 0, fmt.Errorf("无法解析时间 %q：毫秒部分无效", s)
	}

	total := mm*60000 + ss*1000 + msPart
	if total < 0 {
		return 0, fmt.Errorf("无法解析时间 %q：不能为负值", s)
	}
	return total, nil
}
