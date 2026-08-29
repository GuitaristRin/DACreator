package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// ImportLegacyDat 解析旧版 Player_ID.dat（KEY = VALUE 行格式）并转换为 v3 配置。
// 兼容性说明：
//   - 地区键兼容 REGION 与 LOCALE（旧版模板用 LOCALE，GUI 写 REGION）；
//   - 等号前后空格可有可无（旧版文件存在 "VERSION =2.1.1" 这类写法）；
//   - ID 做 NFKC 归一化，兼容全角用户名；
//   - VERSION 等未知键忽略。
func ImportLegacyDat(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取旧版配置 %s: %w", path, err)
	}

	cfg := Default()
	keys := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		keys[strings.ToUpper(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}

	cfg.ID = norm.NFKC.String(keys["ID"])
	cfg.Region = firstNonEmpty(keys["REGION"], keys["LOCALE"])
	cfg.City = keys["CITY"]
	cfg.Store = keys["STORE"]
	cfg.Team = keys["TEAM"]
	if n, err := strconv.Atoi(keys["SEASON"]); err == nil && n > 0 {
		cfg.Season = n
	}
	if n, err := strconv.Atoi(keys["ROUND"]); err == nil && n > 0 {
		cfg.Round = n
	}
	return cfg, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
