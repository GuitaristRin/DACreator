// 应用状态与帧循环：单一状态结构，页面渲染为纯函数（对齐 PezMax-One 模式）。

use sokuou::SpringAnim;

use crate::components;
use crate::pages;
use crate::theme;
use std::path::PathBuf;

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
}

impl DacApp {
    pub fn new(cc: &eframe::CreationContext<'_>) -> Self {
        let font_path = assets_dir().join("font").join("NotoSansCJKsc-Bold.otf");
        theme::setup_fonts(&cc.egui_ctx, Some(font_path));

        let theme_mode = theme::ThemeMode::System;
        theme::set_dark(theme::effective_dark(&cc.egui_ctx, theme_mode));

        Self {
            theme_mode,
            current_section: Section::Home,
            sidebar_open: true,
            sidebar_anim: SpringAnim::new(0.5, 0.825, 1.0),
            sidebar_indicator_anim: SpringAnim::new(0.3, 0.8, 0.0),
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
