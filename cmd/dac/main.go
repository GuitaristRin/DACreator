// Command dac 是 DACreator 的引擎 CLI 入口。
// GUI（gui/）通过 `--json` 事件流消费本命令的全部能力。
package main

import (
	"fmt"
	"os"
)

// version 由构建脚本通过 ldflags 注入：-X main.version=x.y.z
var version = "dev"

const usage = `DACreator v3 引擎 —— 头文字D 激斗成绩工具

用法：
  dac card      [-d 图片目录] [--json]                                     生成简报成绩卡
  dac recordcard [-d 图片目录] [--json]                                    生成记录卡（等级统计+精选成绩）
  dac crawl     [-u 用户名] [-s 赛季] [-d 图片目录] [-c 并发数] [--json]   爬取全部赛道成绩
  dac localcsv  <成绩.csv> [-d 图片目录] [--json]                          本地 CSV 生成表格
  dac history   [-c 赛道] [-n 条数] [--json]                               查询历史记录
  dac config    show [--json] | set --id 名称 … | import <Player_ID.dat>  查看或导入配置
  dac update    check [--json]                                            检查更新
  dac version   [--json]                                                  版本信息

全局约定：
  * 原始 CSV 始终保存到数据目录 raw/ 下（Windows: %APPDATA%/DACreator）。
  * --json 时以 JSON-lines 事件流输出（GUI 消费协议），否则为人读文本。`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "crawl":
		err = cmdCrawl(os.Args[2:])
	case "localcsv":
		err = cmdLocalCSV(os.Args[2:])
	case "card":
		err = cmdCard(os.Args[2:])
	case "history":
		err = cmdHistory(os.Args[2:])
	case "config":
		err = cmdConfig(os.Args[2:])
	case "version":
		err = cmdVersion(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Println(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "未知子命令 %q\n\n%s\n", os.Args[1], usage)
		os.Exit(2)
	}

	if err != nil {
		// 事件流走 stdout（--json 协议），错误细节输出到 stderr
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}
