package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/GuitaristRin/DACreator/internal/az"
	"github.com/GuitaristRin/DACreator/internal/config"
	"github.com/GuitaristRin/DACreator/internal/events"
	"github.com/GuitaristRin/DACreator/internal/model"
	"github.com/GuitaristRin/DACreator/internal/render"
	"github.com/GuitaristRin/DACreator/internal/store"
	"github.com/GuitaristRin/DACreator/internal/update"
	"golang.org/x/text/unicode/norm"
)

func cmdCrawl(args []string) error {
	fs := flag.NewFlagSet("crawl", flag.ExitOnError)
	user := fs.String("u", "", "ArcadeZone 用户名（缺省读配置）")
	season := fs.Int("s", 0, "赛季 1-10（缺省读配置）")
	outDir := fs.String("d", "", "表格图片输出目录（缺省只保存原始 CSV）")
	concurrency := fs.Int("c", az.DefaultConcurrency, "并发请求数")
	jsonMode := fs.Bool("json", false, "以 JSON-lines 事件流输出")
	flags, _ := splitFlagArgs(args, "u", "s", "d", "c")
	if err := fs.Parse(flags); err != nil {
		return err
	}

	emit := events.NewEmitter(os.Stdout, *jsonMode)
	start := time.Now()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if *user != "" {
		cfg.ID = norm.NFKC.String(*user)
	}
	if *season > 0 {
		cfg.Season = *season
	}
	if cfg.ID == "" {
		emit.Error("config", "尚未配置用户名：请运行 dac config import 或在 GUI 中设置")
		return errors.New("尚未配置用户名")
	}
	emit.Log(events.LevelInfo, fmt.Sprintf("目标用户：%s（第 %d 赛季）", cfg.ID, cfg.Season))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client := az.NewClient(cfg.ID, cfg.Season)
	records, err := client.CrawlAll(ctx, emit, *concurrency)
	if err != nil {
		if errors.Is(err, az.ErrNoRecords) {
			emit.Error("notfound", "未找到任何成绩记录：请检查用户名与赛季")
			return errors.New("未找到任何成绩记录")
		}
		if ctx.Err() != nil {
			emit.Error("cancel", "用户中断")
			return errors.New("用户中断")
		}
		emit.Error("network", err.Error())
		return err
	}

	// 原始 CSV 一律保存（数据目录 raw/ 下，文件名与旧版一致）
	rawDir, err := rawDir()
	if err != nil {
		emit.Error("io", err.Error())
		return err
	}
	csvPath := filepath.Join(rawDir, fmt.Sprintf("%s_search.csv", fileTimestamp()))
	if err := writeCSVFile(records, csvPath); err != nil {
		emit.Error("io", err.Error())
		return err
	}
	emit.Log(events.LevelSuccess, "原始 CSV 已保存："+csvPath)

	// 可选：生成表格图片
	pngPath := ""
	if *outDir != "" {
		pngPath, err = renderTable(emit, records, *outDir)
		if err != nil {
			emit.Error("io", err.Error())
			return err
		}
	}

	insertHistory(emit, records, "search")
	emit.Result(csvPath, pngPath, len(records), time.Since(start))
	return nil
}

// cmdCard 生成简报成绩卡：计时赛 Top5 + 回合/名声/车队数据。
func cmdCard(args []string) error {
	fs := flag.NewFlagSet("card", flag.ExitOnError)
	outDir := fs.String("d", "", "成绩卡输出目录（缺省保存到数据目录 raw/card）")
	jsonMode := fs.Bool("json", false, "以 JSON-lines 事件流输出")
	flags, _ := splitFlagArgs(args, "d")
	if err := fs.Parse(flags); err != nil {
		return err
	}

	emit := events.NewEmitter(os.Stdout, *jsonMode)
	start := time.Now()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.ID == "" {
		emit.Error("config", "尚未配置用户名：请运行 dac config set 或在 GUI 中设置")
		return errors.New("尚未配置用户名")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client := az.NewClient(cfg.ID, cfg.Season)
	data, err := client.FetchCardData(ctx, cfg.Team, cfg.Round, emit)
	if err != nil {
		if ctx.Err() != nil {
			emit.Error("cancel", "用户中断")
			return errors.New("用户中断")
		}
		emit.Error("network", err.Error())
		return err
	}

	emit.Progress("card", 80, "渲染成绩卡")
	out := *outDir
	if out == "" {
		raw, err := rawDir()
		if err != nil {
			emit.Error("io", err.Error())
			return err
		}
		out = filepath.Join(raw, "card")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		emit.Error("io", err.Error())
		return err
	}
	pngPath := filepath.Join(out, fmt.Sprintf("race_card_%s.png", imageTimestamp()))
	img, err := render.RenderCard(render.CardInput{
		PlayerID: cfg.ID, Region: cfg.Region, City: cfg.City, Store: cfg.Store,
		Season: cfg.Season, Round: cfg.Round,
		Records: data.Records, RoundScore: data.RoundScore,
		PrideValue: data.PrideValue, TeamName: cfg.Team,
		TeamScore: data.TeamScore, TeamLevel: data.TeamLevel,
	}, render.DefaultCardConfig(config.AssetsDir()))
	if err != nil {
		emit.Error("io", err.Error())
		return err
	}
	if err := render.SavePNG(img, pngPath); err != nil {
		emit.Error("io", err.Error())
		return err
	}
	emit.Log(events.LevelSuccess, "成绩卡已保存："+pngPath)

	insertHistory(emit, data.Records, "card")
	emit.Result("", pngPath, len(data.Records), time.Since(start))
	return nil
}

func cmdLocalCSV(args []string) error {
	fs := flag.NewFlagSet("localcsv", flag.ExitOnError)
	outDir := fs.String("d", "", "表格图片输出目录（缺省只保存原始 CSV 到 raw）")
	jsonMode := fs.Bool("json", false, "以 JSON-lines 事件流输出")
	flags, pos := splitFlagArgs(args, "d")
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if len(pos) < 1 {
		return errors.New("用法：dac localcsv <成绩.csv> [-d 图片目录]")
	}

	emit := events.NewEmitter(os.Stdout, *jsonMode)
	start := time.Now()

	srcPath := pos[0]
	data, err := os.ReadFile(srcPath)
	if err != nil {
		emit.Error("io", fmt.Sprintf("读取 CSV 失败：%v", err))
		return err
	}
	records, err := model.ParseCSV(bytes.NewReader(data))
	if err != nil {
		emit.Error("parse", err.Error())
		return err
	}
	emit.Log(events.LevelInfo, fmt.Sprintf("已读取 %d 条记录：%s", len(records), srcPath))

	rawDirPath, err := rawDir()
	if err != nil {
		emit.Error("io", err.Error())
		return err
	}
	csvPath := filepath.Join(rawDirPath, fmt.Sprintf("%s_localcsv.csv", fileTimestamp()))
	if err := writeCSVFile(records, csvPath); err != nil {
		emit.Error("io", err.Error())
		return err
	}
	emit.Log(events.LevelSuccess, "原始 CSV 已保存："+csvPath)

	pngPath := ""
	if *outDir != "" {
		pngPath, err = renderTable(emit, records, *outDir)
		if err != nil {
			emit.Error("io", err.Error())
			return err
		}
	}

	insertHistory(emit, records, "localcsv")
	emit.Result(csvPath, pngPath, len(records), time.Since(start))
	return nil
}

func cmdHistory(args []string) error {
	fs := flag.NewFlagSet("history", flag.ExitOnError)
	course := fs.String("c", "", "按赛道筛选")
	limit := fs.Int("n", 100, "最大条数")
	jsonMode := fs.Bool("json", false, "以 JSON 输出")
	fs.Parse(args)

	dbPath, err := historyPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	records, err := s.History(*course, *limit)
	if err != nil {
		return err
	}
	if *jsonMode {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"records": records})
	}
	fmt.Printf("共 %d 条记录\n", len(records))
	for _, r := range records {
		national := r.NationalRank
		if national != "" {
			national += "位"
		}
		fmt.Printf("  %s | %s | %s | %s | %s | %s | %s | %s\n",
			r.RecordDate, r.Course, r.Direction, r.TimeStr, r.Rank, r.Car, national, r.CreatedAt)
	}
	return nil
}

func cmdConfig(args []string) error {
	if len(args) == 0 {
		return cmdConfigShow(nil)
	}
	switch args[0] {
	case "show":
		return cmdConfigShow(args[1:])
	case "import":
		if len(args) < 2 {
			return errors.New("用法：dac config import <Player_ID.dat 路径>")
		}
		return cmdConfigImport(args[1])
	case "set":
		return cmdConfigSet(args[1:])
	default:
		return fmt.Errorf("未知 config 子命令 %q（支持 show / import）", args[0])
	}
}

func cmdConfigShow(args []string) error {
	fs := flag.NewFlagSet("config show", flag.ExitOnError)
	jsonMode := fs.Bool("json", false, "以 JSON 输出")
	fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if *jsonMode {
		return json.NewEncoder(os.Stdout).Encode(cfg)
	}
	path, _ := config.Path()
	fmt.Printf("配置文件：%s\n", path)
	fmt.Printf("  ID     = %s\n", cfg.ID)
	fmt.Printf("  REGION = %s\n", cfg.Region)
	fmt.Printf("  CITY   = %s\n", cfg.City)
	fmt.Printf("  STORE  = %s\n", cfg.Store)
	fmt.Printf("  TEAM   = %s\n", cfg.Team)
	fmt.Printf("  SEASON = %d\n", cfg.Season)
	fmt.Printf("  ROUND  = %d\n", cfg.Round)
	return nil
}

func cmdConfigImport(datPath string) error {
	cfg, err := config.ImportLegacyDat(datPath)
	if err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Println("✅ 旧版配置已导入：", configDisplayPath())
	fmt.Printf("  ID = %s，赛季 = %d\n", cfg.ID, cfg.Season)
	return nil
}

func configDisplayPath() string {
	p, err := config.Path()
	if err != nil {
		return "config.toml"
	}
	return p
}

// cmdConfigSet 增量更新配置（GUI 设置页的写入通路）。
// 只提供 flag 的字段会被更新，未提供的字段保持原值。
func cmdConfigSet(args []string) error {
	fs := flag.NewFlagSet("config set", flag.ExitOnError)
	id := fs.String("id", "", "ArcadeZone 用户名")
	region := fs.String("region", "", "店铺所在地区")
	city := fs.String("city", "", "店铺所在城市")
	store := fs.String("store", "", "店铺名")
	team := fs.String("team", "", "车队名")
	season := fs.Int("season", 0, "赛季 1-10")
	round := fs.Int("round", 0, "回合 1-10")
	jsonMode := fs.Bool("json", false, "以 JSON 输出更新后的配置")
	flags, _ := splitFlagArgs(args, "id", "region", "city", "store", "team", "season", "round")
	if err := fs.Parse(flags); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if *id != "" {
		cfg.ID = *id
	}
	if *region != "" {
		cfg.Region = *region
	}
	if *city != "" {
		cfg.City = *city
	}
	if *store != "" {
		cfg.Store = *store
	}
	if *team != "" {
		cfg.Team = *team
	}
	if *season > 0 {
		cfg.Season = *season
	}
	if *round > 0 {
		cfg.Round = *round
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	if *jsonMode {
		return json.NewEncoder(os.Stdout).Encode(cfg)
	}
	fmt.Println("✅ 配置已更新")
	return nil
}

// cmdUpdate 检查 GitHub Releases 上的新版本。
func cmdUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	jsonMode := fs.Bool("json", false, "以 JSON 输出检查结果")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rel, hasUpdate, err := update.CheckUpdate(context.Background(), version)
	if err != nil {
		emit := events.NewEmitter(os.Stdout, *jsonMode)
		emit.Error("network", err.Error())
		return err
	}
	assetName, assetURL := "", ""
	if len(rel.Assets) > 0 {
		assetName = rel.Assets[0].Name
		assetURL = rel.Assets[0].BrowserDownloadURL
	}
	if *jsonMode {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"current":    version,
			"latest":     rel.TagName,
			"has_update": hasUpdate,
			"notes":      rel.Body,
			"asset_name": assetName,
			"asset_url":  assetURL,
		})
	}
	if hasUpdate {
		fmt.Printf("🎉 发现新版本 %s（当前 %s）\n更新说明：\n%s\n", rel.TagName, version, rel.Body)
		if assetURL != "" {
			fmt.Printf("下载：%s\n", assetURL)
		}
	} else {
		fmt.Printf("✅ 已是最新版本（%s）\n", version)
	}
	return nil
}

func cmdVersion(args []string) error {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	jsonMode := fs.Bool("json", false, "以 JSON 输出")
	fs.Parse(args)
	if *jsonMode {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"version": version})
	}
	fmt.Printf("DACreator 引擎 %s\n", version)
	return nil
}

// ---------- 共用辅助 ----------

// splitFlagArgs 把参数重排为「flags 在前、位置参数在后」。
// Go 的 flag 包遇到首个位置参数即停止解析，而用户习惯写 `dac localcsv 文件.csv -d 目录`，
// 这里按已知带值 flag 的约定把序列还原，保证两种写法都可用。
func splitFlagArgs(args []string, valueFlags ...string) (flags, pos []string) {
	takesValue := map[string]bool{}
	for _, n := range valueFlags {
		takesValue[n] = true
	}
	expect := false
	for _, a := range args {
		switch {
		case expect:
			flags = append(flags, a)
			expect = false
		case strings.HasPrefix(a, "-") && a != "-" && a != "--":
			flags = append(flags, a)
			name := strings.TrimLeft(a, "-")
			if i := strings.IndexByte(name, '='); i >= 0 {
				name = name[:i] // -d=out 形式自带值
			}
			expect = takesValue[name]
		default:
			pos = append(pos, a)
		}
	}
	return flags, pos
}

func rawDir() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "raw")
	if err := os.MkdirAll(p, 0o755); err != nil {
		return "", fmt.Errorf("创建 raw 目录 %s: %w", p, err)
	}
	return p, nil
}

func historyPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.db"), nil
}

func fileTimestamp() string { return time.Now().Format("200601021504") }

func imageTimestamp() string { return time.Now().Format("20060102_150405") }

func writeCSVFile(records []model.Record, path string) error {
	data, err := model.RecordToCSVBytes(records)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func renderTable(emit *events.Emitter, records []model.Record, outDir string) (string, error) {
	emit.Progress("render", 0, "开始渲染表格图片")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("创建输出目录 %s: %w", outDir, err)
	}
	img, err := render.RenderTable(records, render.DefaultConfig(config.AssetsDir()))
	if err != nil {
		return "", err
	}
	pngPath := filepath.Join(outDir, fmt.Sprintf("DAC成绩表_%s.png", imageTimestamp()))
	if err := render.SavePNG(img, pngPath); err != nil {
		return "", err
	}
	emit.Progress("render", 100, "表格图片生成完成")
	emit.Log(events.LevelSuccess, "表格图片已保存："+pngPath)
	return pngPath, nil
}

func insertHistory(emit *events.Emitter, records []model.Record, source string) {
	dbPath, err := historyPath()
	if err != nil {
		emit.Log(events.LevelWarning, "历史数据库不可用："+err.Error())
		return
	}
	s, err := store.Open(dbPath)
	if err != nil {
		emit.Log(events.LevelWarning, "历史数据库打开失败："+err.Error())
		return
	}
	defer s.Close()
	n, err := s.InsertRecords(records, source)
	if err != nil {
		emit.Log(events.LevelWarning, "历史记录写入失败："+err.Error())
		return
	}
	emit.Log(events.LevelSuccess, fmt.Sprintf("历史数据库已写入 %d 条新记录", n))
}
