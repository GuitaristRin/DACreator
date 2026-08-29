// 应用状态与帧循环：单一状态结构，页面渲染为纯函数（对齐 PezMax-One 模式）。

use sokuou::SpringAnim;
use std::path::PathBuf;
use std::time::Instant;

use crate::components;
use crate::engine::{self, EngineOutput, RunningTask};
use crate::pages;
use crate::pages::home::{HomeMode, LogLine, TaskResult};
use crate::theme;

/// 顶层导航分区。
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Section {
    Home,
    Records,
    Settings,
    About,
}

impl Section {
    pub const ALL: [Section; 4] = [
        Section::Home,
        Section::Records,
        Section::Settings,
        Section::About,
    ];

    pub fn icon(self) -> &'static str {
        match self {
            Section::Home => "🏠",
            Section::Records => "📋",
            Section::Settings => "⚙️",
            Section::About => "ℹ️",
        }
    }

    pub fn title(self) -> &'static str {
        match self {
            Section::Home => "主页",
            Section::Records => "数据",
            Section::Settings => "设置",
            Section::About => "关于",
        }
    }

    /// 侧边栏指示器的目标索引。
    pub fn index(self) -> usize {
        Section::ALL.iter().position(|s| *s == self).unwrap_or(0)
    }
}

/// 单一状态结构：所有应用状态挂在这里。
pub struct DacApp {
    pub theme_mode: theme::ThemeMode,

    pub current_section: Section,
    // sidebar_anim: 0.0 = 折叠(54px) / 1.0 = 展开(200px)
    pub sidebar_open: bool,
    pub sidebar_anim: SpringAnim,
    // sidebar_indicator_anim: 值为当前高亮 Section 的索引（0-3），弹簧插值
    pub sidebar_indicator_anim: SpringAnim,

    // ── 引擎与任务 ─────────────────────────────────────────────
    pub engine_exe: Option<PathBuf>,
    pub task: Option<RunningTask>,
    pub task_running: bool,
    pub task_started_at: Option<Instant>,
    pub progress: Option<(String, i32, String)>,
    pub logs: Vec<LogLine>,
    pub last_result: Option<TaskResult>,
    pub last_error: Option<String>,

    // ── 主页表单 ───────────────────────────────────────────────
    pub home_mode: HomeMode,
    pub csv_path: Option<String>,
    pub out_dir: Option<String>,

    app_start: Instant,
}

impl DacApp {
    pub fn new(cc: &eframe::CreationContext<'_>) -> Self {
        let font_path = assets_dir().join("font").join("NotoSansCJKsc-Bold.otf");
        theme::setup_fonts(&cc.egui_ctx, Some(font_path));

        let theme_mode = theme::ThemeMode::System;
        theme::set_dark(theme::effective_dark(&cc.egui_ctx, theme_mode));

        let engine_exe = engine::resolve_engine_exe();

        Self {
            theme_mode,
            current_section: Section::Home,
            sidebar_open: true,
            sidebar_anim: SpringAnim::new(0.5, 0.825, 1.0),
            sidebar_indicator_anim: SpringAnim::new(0.3, 0.8, 0.0),
            engine_exe,
            task: None,
            task_running: false,
            task_started_at: None,
            progress: None,
            logs: Vec::new(),
            last_result: None,
            last_error: None,
            home_mode: HomeMode::default(),
            csv_path: None,
            out_dir: None,
            app_start: Instant::now(),
        }
    }

    /// 切换顶层分区（侧边栏滑块随之弹簧插值）。
    pub fn navigate(&mut self, section: Section) {
        if self.current_section == section {
            return;
        }
        self.current_section = section;
        self.sidebar_indicator_anim
            .set_target(section.index() as f64);
    }

    fn push_log(&mut self, level: &str, msg: impl Into<String>) {
        let elapsed = self
            .task_started_at
            .unwrap_or(self.app_start)
            .elapsed()
            .as_secs_f64();
        self.logs.push(LogLine {
            elapsed_secs: elapsed,
            level: level.to_owned(),
            msg: msg.into(),
        });
    }

    /// 启动主页任务（成绩爬取 / 本地 CSV）。
    pub fn start_home_task(&mut self) {
        if self.task_running {
            return;
        }
        self.logs.clear();
        self.last_error = None;
        self.last_result = None;
        self.progress = None;
        self.task_started_at = Some(Instant::now());

        let Some(exe) = self.engine_exe.clone() else {
            self.push_log(
                "error",
                "未找到引擎程序 dac.exe，可用环境变量 DACREATOR_ENGINE 指定路径",
            );
            return;
        };

        let (mut args, label) = match self.home_mode {
            HomeMode::Crawl => (vec!["crawl".to_owned(), "--json".to_owned()], "成绩爬取"),
            HomeMode::LocalCsv => {
                let Some(csv) = self.csv_path.clone() else {
                    self.push_log("error", "请先选择 CSV 文件");
                    return;
                };
                (
                    vec!["localcsv".to_owned(), csv, "--json".to_owned()],
                    "本地 CSV",
                )
            }
        };
        if let Some(dir) = self.out_dir.clone() {
            args.extend(["-d".to_owned(), dir]);
        }

        match engine::spawn_engine(&exe, &args, label) {
            Ok(task) => {
                self.task = Some(task);
                self.task_running = true;
                self.push_log("info", format!("{label}任务已启动"));
            }
            Err(e) => self.push_log("error", format!("引擎启动失败：{e}")),
        }
    }

    /// 停止当前任务。
    pub fn stop_home_task(&mut self) {
        if let Some(task) = &self.task {
            task.kill();
            self.push_log("warning", "已请求停止引擎");
        }
    }

    /// 每帧轮询引擎事件（非阻塞）。
    fn poll_engine(&mut self, ctx: &egui::Context) {
        let mut events = Vec::new();
        if let Some(task) = &self.task {
            while let Some(ev) = task.try_recv() {
                events.push(ev);
            }
        }
        if events.is_empty() {
            return;
        }

        let mut exited = false;
        for ev in events {
            match ev {
                EngineOutput::Progress { stage, pct, detail } => {
                    self.progress = Some((stage_label(&stage), pct, detail));
                }
                EngineOutput::Log { level, msg } => self.push_log(&level, msg),
                EngineOutput::Result {
                    csv_path,
                    png_path,
                    records,
                    elapsed_ms,
                } => {
                    self.last_result = Some(TaskResult {
                        csv_path,
                        png_path: if png_path.is_empty() {
                            None
                        } else {
                            Some(png_path)
                        },
                        records,
                        elapsed_ms,
                    });
                }
                EngineOutput::Error { code, msg } => {
                    self.last_error = Some(msg.clone());
                    self.push_log("error", format!("[{code}] {msg}"));
                }
                EngineOutput::Exited { success } => {
                    exited = true;
                    if success {
                        self.push_log("success", "任务结束");
                    } else if self.last_error.is_none() {
                        self.push_log("error", "引擎异常退出");
                    }
                }
            }
        }
        if exited {
            self.task = None;
            self.task_running = false;
            self.progress = None;
        }
        ctx.request_repaint();
    }
}

fn stage_label(stage: &str) -> String {
    match stage {
        "crawl" => "爬取".to_owned(),
        "render" => "渲染".to_owned(),
        "save" => "保存".to_owned(),
        "card" => "成绩卡".to_owned(),
        other => other.to_owned(),
    }
}

/// 解析随包资产目录：环境变量 → 可执行文件同级 → 当前目录（开发兜底）。
pub fn assets_dir() -> PathBuf {
    if let Ok(v) = std::env::var("DACREATOR_ASSETS") {
        if !v.is_empty() {
            return PathBuf::from(v);
        }
    }
    if let Ok(exe) = std::env::current_exe() {
        if let Some(dir) = exe.parent() {
            let p = dir.join("assets");
            if p.is_dir() {
                return p;
            }
        }
    }
    PathBuf::from("assets")
}

impl eframe::App for DacApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        let dt = ctx.input(|i| i.stable_dt) as f64;

        // ── 引擎事件轮询 ──────────────────────────────────────────
        self.poll_engine(ctx);
        if self.task_running {
            ctx.request_repaint();
        }

        // ── 动画推进（Sokuou 统一驱动）────────────────────────────
        self.sidebar_anim.update(dt);
        self.sidebar_indicator_anim.update(dt);
        theme::update_dark_transition(dt);
        theme::update_accent_transition(dt);
        let animating = !self.sidebar_anim.is_steady()
            || !self.sidebar_indicator_anim.is_steady()
            || theme::is_dark_transitioning()
            || theme::is_transitioning();
        if animating {
            ctx.request_repaint();
        }

        // ── System 模式跟随运行时主题变化（过渡期间不重复触发）────
        if self.theme_mode == theme::ThemeMode::System && !theme::is_dark_transitioning() {
            let now_dark = theme::effective_dark(ctx, self.theme_mode);
            if now_dark != theme::is_dark() {
                theme::start_dark_transition(now_dark);
                theme::set_dark(now_dark);
            }
        }
        theme::apply_metro_theme(ctx);

        // ── 布局：左侧栏 + 顶栏 + 中央页面 ────────────────────────
        components::sidebar::render(self, ctx);
        components::topbar::render(self, ctx);
        pages::dispatch(self, ctx);
    }
}
