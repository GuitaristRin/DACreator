# AGENT.md

本文件约束 AI 助手（Agent）在本仓库工作时的行为规范与项目架构约定。任何代码改动、提交、文档撰写都必须遵守本文。

## 项目定位

DACreator v3：为《头文字D 激斗》玩家爬取 ArcadeZone 计时赛成绩、生成可视化表格与成绩卡、记录历史数据的桌面工具。

v3 是对旧版 Python 实现（CLI + PyQt5 GUI 两个仓库）的**全量替代性重写**，采用 **Go + Rust 双语言架构**：

- **Go 引擎（`dac`）**：爬虫、数据管线、表格/成绩卡渲染、SQLite 历史、自更新、CLI。
- **Rust GUI（`DACreator`）**：egui/eframe 桌面壳，Metro Design + Sokuou 动画，风格基准为 PezMax-One。
- 旧 Python 代码只作参考（`legacy/`），v3.0.0 发布后删除。旧版 GUI 仓库（DACreator-GUI）归档，不再维护。

## 仓库布局（Monorepo）

```
DACreator/                     ← 仓库根 = Go module 根
├── AGENT.md                   ← 本文件
├── README.md
├── go.mod                     ← module github.com/GuitaristRin/DACreator
├── cmd/dac/main.go            ← 引擎 CLI 入口（二合一的 CLI 面）
├── internal/
│   ├── model/                 ← Record 等核心类型、CSV schema（编解码唯一实现）
│   ├── config/                ← config.toml 读写 + 旧版 Player_ID.dat 导入
│   ├── az/                    ← ArcadeZone 客户端（CSRF、搜索 API、并发、NFKC、eval_id）
│   ├── render/                ← 表格图 / 成绩卡渲染（PIL 逻辑的 Go 重写）
│   ├── store/                 ← SQLite 历史（modernc.org/sqlite，纯 Go 无 cgo）
│   ├── update/                ← 自更新（GitHub Releases API）
│   └── events/                ← JSON-lines 事件协议（GUI↔引擎 的唯一契约）
├── gui/                       ← Rust crate（egui/eframe 0.31）
│   ├── Cargo.toml
│   ├── sokuou/                ← git submodule → github.com/GuitaristRin/Sokuou
│   └── src/
│       ├── main.rs            ← eframe 入口、窗口配置
│       ├── app.rs             ← 单一状态结构 DacApp + 路由（对齐 PezMax-One 模式）
│       ├── engine/            ← EngineHandle：引擎子进程管理 + JSON-lines 解析
│       ├── theme/             ← Metro Design（自 PezMax-One 移植，含深浅色/强调色过渡）
│       ├── components/        ← sidebar / topbar / toast / spinner / 进度组件
│       └── pages/             ← home / records / settings / about
├── assets/                    ← 字体（仅 OFL 许可）、rank 徽章图、lang 多语言文件、rank.csv
├── scripts/                   ← 构建 / 交叉编译 / WiX 打包脚本
└── legacy/                    ← 旧 Python 实现（只读参考，v3.0.0 后删除）
```

## 语言分工与边界

**分工依据：性能优先，谁合适谁上。**

| 组件 | 语言 | 依据 |
|------|------|------|
| GUI 与动画 | **Rust**（egui + Sokuou） | GPU 驱动帧循环，动画体验已被 PezMax-One 验证；PyQt5 无法比拟 |
| 爬虫 / JSON / 数据管线 | **Go** | 网络 IO 密集，Go 与 Rust 吞吐等价；goroutine 并发抓取 48 赛道 + 接口收敛表达力 + 编译迭代速度 |
| 表格图 / 成绩卡渲染 | **Go** | 单次毫秒级计算，两者无差别；放引擎保证 CLI 与 GUI 同一渲染通路 |
| 历史存储 | **Go**（SQLite） | 单一数据通路要求存储归引擎 |
| 自更新 | **Go** | 版本判定/下载与引擎版本同源 |

**边界规则：**

1. GUI **不直接**访问网络爬取、SQLite、配置文件——一切数据操作通过调用引擎子进程完成（EngineHandle）。GUI 侧只有 UI 状态与引擎事件缓存。
2. 引擎对 GUI 只暴露**一个契约**：`internal/events` 定义的 JSON-lines 事件流。契约变更必须先改 `events` 包并在提交说明中显式注明。
3. Go 侧用小接口收敛能力（`Crawler` / `Renderer` / `Store` / `Updater`），`internal/*` 之间只依赖接口与 `model`，禁止跨包摸私有实现。
4. CLI 与 GUI 是引擎的两个平等消费者：`dac <子命令>` 输出人类可读文本；`dac --json <子命令>` 输出 JSON-lines。GUI 固定走 `--json`。

事件流草案（以实际实现为准，改动需同步本节示例）：

```json
{"type":"progress","stage":"crawl","pct":42,"detail":"赛道 12/48"}
{"type":"log","level":"info","msg":"..."}
{"type":"result","csv_path":"...","png_path":"...","records":123,"elapsed_ms":5230}
{"type":"error","code":"network","msg":"..."}
```

**二合一形态：** `DACreator.exe` 无参数启动 GUI；带子命令参数时透传给引擎执行（打包时内嵌引擎，首次运行释放到数据目录）。M5 打包阶段实现内嵌；在此之前 GUI 与 `dac.exe` 成对分发，接口不变。

## 依赖与 Submodule

- **Sokuou 动画引擎以 git submodule 引入**：`gui/sokuou` → `https://github.com/GuitaristRin/Sokuou`，`gui/Cargo.toml` 以 path 依赖引用（`sokuou = { path = "sokuou" }`）。
- 克隆/拉取后必须执行 `git submodule update --init --recursive`；CI 与构建脚本需显式处理 submodule。
- Agent 在本仓库内**只升级 submodule 指针**，以独立 `chore:` 提交完成（如 `chore: 升级 Sokuou submodule 指针至 v0.1.1。`）；**禁止在本仓库内改动 submodule 内部文件**——Sokuou 本体的改动去其自身仓库提交，且同样由用户手动推送。
- 其余依赖：Go 侧仅允许 `modernc.org/sqlite`、`golang.org/x/image` 等纯 Go 依赖（禁止 cgo）；Rust 侧以 PezMax-One 的依赖清单为基准，TLS 一律 `rustls`。

## 构建与测试

```bash
git clone --recursive https://github.com/GuitaristRin/DACreator.git   # submodule 必须带上
go build ./... && go vet ./... && go test ./...    # 引擎
cd gui && cargo fmt --check && cargo clippy && cargo test   # GUI
go run ./cmd/dac --spider                          # 本地跑引擎 CLI
```

- 提交前对应语言的 lint + test 必须全绿；未覆盖的解析逻辑（时间解析、NFKC 过滤、评级映射、CSV 编解码）必须有表驱动测试。
- 目标平台：Windows x64 优先；Go 侧保持 `GOOS=darwin/linux` 可编译（不做 GUI 承诺）。

## 提交规范（强制）

1. **格式：`前缀: 正文。`** 正文用中文，**必须以句号 `。` 结尾**。前缀与正文之间是 `: `（冒号 + 空格）。
2. 前缀集合：`feat` `fix` `perf` `refactor` `test` `docs` `chore` `build` `ci`。
3. **一个 section 一次 commit**：一个功能块/模块/独立改动点 = 一个 commit。禁止把多个不相关 block 塞进同一提交——这是为了回溯历史时每个提交语义完整、可单独 revert。
4. **Agent 永远不执行 `git push`**：推送一律由用户手动完成。同样禁止 `git push --force`、禁止改写已推送历史。
5. 每次提交前确认构建通过；测试与所测功能属于同一 section，随该 section 的提交走，不单独拆测试提交（除非是纯补测试的 section）。
6. 示例：
   - ✅ `feat: 新增 ArcadeZone 搜索爬虫与服务器评级映射。`
   - ✅ `chore: 迁移旧版 Python 实现至 legacy 目录。`
   - ✅ `fix: 修复搜索结果未按 NFKC 归一化过滤用户名的问题。`
   - ❌ `feat: 新增爬虫并顺便改了主题和 README`（多 section 混提交、无句号）
   - ❌ `feat: added crawler`（英文正文、无句号）

## 代码规范

**Go：**
- `gofmt` + `go vet` 干净；错误用 `%w` 包装并附上下文（`fmt.Errorf("加载配置: %w", err)`）。
- 包级配置一律注入，禁止包内直接读全局状态；路径解析基于 `os.Executable()` / `dirs` 数据目录，**禁止依赖进程 CWD 的相对路径**。
- 解析类逻辑（时间字符串、CSV、评级）必须表驱动测试。

**Rust：**
- `rustfmt` + `clippy` 无警告；错误用 `thiserror` 定义错误枚举，UI 层用 `anyhow` 汇聚。
- 对齐 PezMax-One 架构模式：单一状态结构、页面为纯函数、异步经 tokio + oneshot 回帧循环、`AsyncData<T>` 包装加载态。
- **所有动画必须走 Sokuou**（`SpringAnim` 位移/尺寸、`Progress` 透明度/颜色、`MetroAnim` 主题过渡），禁止裸计时器或 egui 内建动画；动画实例挂 `DacApp` 字段，render 函数内禁止 `set_target`。

## 质量红线

1. **等级唯一来源是服务器 `eval_id`**（映射表见 `internal/az`）。本地 `assets/rank.csv` 仅作成绩卡展示参考，禁止参与等级判定。
2. 搜索结果必须 **NFKC 归一化后精确过滤目标用户名**，防止子串误命中。
3. 搜索接口只请求第 1 页（翻页重复），除非实测推翻此结论。
4. 字体资产只允许 OFL 许可（Noto 系 / 思源系），禁止再分发微软雅黑、Yu Gothic 等系统字体。
5. 版本号单一来源：Go ldflags 注入，`dac version --json` 与 GUI 关于页都从引擎取。禁止再出现"从配置文件读版本号"的设计。
6. CSV schema（`コース,ルート,タイム,タイム評価,記録車種,全国順位,記録日`）保持不变，保证用户既有 CSV 与 `raw/` 历史文件可用。
7. 用户数据目录由 `dirs` 决定（Windows: `%APPDATA%/DACreator/`），旧版 `Player_ID.dat` / `dacreator_history.db` 通过导入/迁移兼容，不向后兼容旧路径行为。
8. 引擎事件消息不内嵌 UI 文案；GUI 依据事件类型与 `assets/lang/*.lang` 键做本地化，自由文本日志原样透传。

## 免责声明约束

项目仅供学习参考，严禁商业用途。README、关于页、安装包必须保留此声明与 MIT 许可证。
