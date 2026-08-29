// Package config 管理 DACreator 用户配置。
// v3 配置存放于平台数据目录（Windows: %APPDATA%/DACreator/config.toml），
// 并提供旧版 Player_ID.dat 的导入。
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/unicode/norm"
)

// Config 是用户配置。字段与旧版 Player_ID.dat 键一一对应（键名统一为 REGION）。
type Config struct {
	ID     string `toml:"id"`     // ArcadeZone 用户名
	Region string `toml:"region"` // 店铺所在地区（旧版模板键名为 LOCALE）
	City   string `toml:"city"`   // 店铺所在城市
	Store  string `toml:"store"`  // 店铺名
	Team   string `toml:"team"`   // 车队名
	Season int    `toml:"season"` // 赛季 1-10
	Round  int    `toml:"round"`  // 回合 1-10
}

// Default 返回默认配置（与旧版默认值一致）。
func Default() Config {
	return Config{Season: 5, Round: 1}
}

// Dir 返回用户数据目录（跨平台）。
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("解析用户数据目录: %w", err)
	}
	return filepath.Join(base, "DACreator"), nil
}

// Path 返回配置文件路径。
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load 读取配置；文件不存在时返回默认配置（不视为错误）。
func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("读取配置 %s: %w", path, err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("解析配置 %s: %w", path, err)
	}
	if cfg.Season <= 0 {
		cfg.Season = Default().Season
	}
	if cfg.Round <= 0 {
		cfg.Round = Default().Round
	}
	cfg.ID = norm.NFKC.String(cfg.ID)
	return cfg, nil
}

// Save 将配置写入数据目录。
func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建数据目录 %s: %w", dir, err)
	}
	cfg.ID = norm.NFKC.String(cfg.ID)
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("编码配置: %w", err)
	}
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("写入配置 %s: %w", path, err)
	}
	return nil
}
