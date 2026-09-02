# DACreator

为《头文字D 激斗》玩家打造的成绩工具：自动爬取 ArcadeZone 计时赛成绩，生成可视化表格与简报成绩卡，并记录历史数据追踪进步。

v3 是 Go + Rust 的全量重写：引擎（爬取/渲染/存储/更新）使用 Go，界面使用 Rust/egui + Metro Design + Sokuou 动画引擎。**一个程序两种用法**——`DACreator.exe` 双击打开图形界面，命令行传参则直接调用引擎。

B 站演示视频：<https://www.bilibili.com/video/BV13SFWzTEnv/>

---

## 下载使用

从 [Releases](https://github.com/GuitaristRin/DACreator/releases) 下载 `DACreator_v*._Windows_x64.zip`，解压即用：

| 文件 | 说明 |
|------|------|
| `DACreator.exe` | 图形界面（双击启动）；带命令行参数时等价 CLI |
| `dac.exe` | 纯 CLI 引擎（脚本调用推荐） |
| `assets/` | 字体、等级徽章、成绩卡模板等（必须与 exe 同级） |

首次使用：打开 **设置** 页填写用户名（ArcadeZone ID）、地区、城市、店铺、车队、赛季与回合，或在设置页从旧版 `Player_ID.dat` 一键导入。

## 功能

- **成绩爬取**：并发抓取全部 48 条计时赛赛道，等级与全国排名取自服务器 `eval_id`（不再依赖本地推算）
- **本地 CSV**：从既有成绩 CSV 生成表格（格式向下兼容旧版）
- **成绩卡**：Top5 记录 + 回合分数 + 名声 + 车队信息一键出图
- **历史记录**：SQLite 自动去重入库，数据页按赛道筛选
- **自动更新**：基于 GitHub Releases 检查新版本
- **重名处理**：搜索结果按 NFKC 归一化精确过滤，同名玩家成绩全部保留

## 命令行

```bash
dac crawl     [-u 用户名] [-s 赛季] [-d 图片目录] [--json]   # 爬取全部赛道
dac localcsv  <成绩.csv> [-d 图片目录] [--json]              # 本地 CSV 生成表格
dac card      [-d 图片目录] [--json]                        # 生成简报成绩卡
dac history   [-c 赛道] [-n 条数] [--json]                  # 查询历史
dac config    show | set --id 名称 … | import <Player_ID.dat>
dac update    check                                         # 检查更新
dac version   [--json]
```

- `--json` 输出 JSON-lines 事件流（进度/日志/结果），供脚本或 GUI 消费；缺省为人读文本
- 原始 CSV 自动保存到数据目录 `raw/` 下（Windows：`%APPDATA%/DACreator`）
- `DACreator.exe` 支持同样的参数：`DACreator.exe crawl -d ./out`

## 数据格式

CSV 列与旧版保持一致（含/不含「全国順位」列均可读取）：

```csv
コース,ルート,タイム,タイム評価,記録車種,全国順位,記録日
秋名湖,左周り,2:27.760,EXPERT,CIVIC TYPE R (FL5) [HC],255,2026/01/19
```

时间列兼容 `2:27.760` 与 `2'27"760` 两种写法，输出统一为规范格式。

## 从旧版迁移

| 旧版 | v3 |
|------|----|
| `Player_ID.dat` | 设置页「从旧版导入」或 `dac config import`；配置存于 `%APPDATA%/DACreator/config.toml` |
| `dacreator_history.db` | 历史库沿用同一表结构，把旧文件复制到 `%APPDATA%/DACreator/history.db` 即可 |
| `raw/` 成绩 CSV | `dac localcsv` 直接可读，schema 不变 |

## 开发者

```bash
git clone --recursive https://github.com/GuitaristRin/DACreator.git   # Sokuou 为 submodule
# 引擎（Go 1.22+）
go build ./... && go test ./...
# 界面（Rust 1.80+）
cargo build --manifest-path gui/Cargo.toml
# 一键打包（含内嵌引擎的便携 zip）
scripts/build.ps1 [-Version 3.0.0]
```

架构与协作规范见 [AGENT.md](AGENT.md)：Go 引擎 + Rust GUI，经 JSON-lines 事件流解耦。

## 注意事项

1. 等级与排名以 ArcadeZone 服务器返回为准；`assets/rank.csv` 仅供成绩卡展示参考。
2. 回合 ID 从排行榜页面内嵌的官方映射（`roundsBySeason`）动态解析，官方调整赛季/回合后自动跟随；仅当页面不可用或结构大改时回退到内置映射（Season 5 实测值），此时日志会给出警告。
3. 内置字体为 Noto Sans CJK（OFL 许可）。
4. 历史数据库永久保存，可手动备份或删除 `%APPDATA%/DACreator/history.db` 重置。

## 免责声明

此程序仅供学习参考，**严禁用于商业用途**！

## 许可证

MIT License © 2026 GuitaristRin
