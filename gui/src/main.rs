//! DACreator GUI（egui/eframe）。
//! v3 架构：GUI 只负责界面与动画，数据操作全部通过引擎子进程（cmd/dac）完成。

mod theme;

use std::path::PathBuf;

use eframe::egui;

fn main() -> eframe::Result {
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default().with_inner_size([1200.0, 800.0]),
        ..Default::default()
    };
    eframe::run_native(
        "DACreator",
        options,
        Box::new(|cc| Ok(Box::new(DacApp::new(cc)))),
    )
}

/// 解析随包资产目录：环境变量 → 可执行文件同级 → 当前目录（开发兜底）。
fn assets_dir() -> PathBuf {
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

/// 单一状态结构：所有应用状态挂在这里，页面渲染为纯函数（对齐 PezMax-One 模式）。
struct DacApp {
    theme_mode: theme::ThemeMode,
}

impl DacApp {
    fn new(cc: &eframe::CreationContext<'_>) -> Self {
        let font_path = assets_dir().join("font").join("NotoSansCJKsc-Bold.otf");
        theme::setup_fonts(&cc.egui_ctx, Some(font_path));

        let theme_mode = theme::ThemeMode::System;
        theme::set_dark(theme::effective_dark(&cc.egui_ctx, theme_mode));

        Self { theme_mode }
    }
}

impl eframe::App for DacApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        let dt = ctx.input(|i| i.stable_dt) as f64;

        // 每帧推进主题过渡；动画未稳态时请求重绘
        theme::update_dark_transition(dt);
        theme::update_accent_transition(dt);
        if theme::is_dark_transitioning() || theme::is_transitioning() {
            ctx.request_repaint();
        }

        // System 模式跟随运行时主题变化（避免与过渡动画打架，过渡期间不重复触发）
        if self.theme_mode == theme::ThemeMode::System && !theme::is_dark_transitioning() {
            let now_dark = theme::effective_dark(ctx, self.theme_mode);
            if now_dark != theme::is_dark() {
                theme::start_dark_transition(now_dark);
                theme::set_dark(now_dark);
            }
        }

        theme::apply_metro_theme(ctx);

        egui::CentralPanel::default().show(ctx, |ui| {
            ui.heading("DACreator v3");
            ui.label("引擎与界面重构进行中…");
            ui.add_space(12.0);
            ui.horizontal(|ui| {
                if ui.button("切换深浅色").clicked() {
                    let target = !theme::is_dark();
                    // 手动切换后脱离 System 跟随
                    self.theme_mode = if target {
                        theme::ThemeMode::Dark
                    } else {
                        theme::ThemeMode::Light
                    };
                    theme::start_dark_transition(target);
                    theme::set_dark(target);
                }
                for (i, preset) in theme::ACCENT_PRESETS.iter().enumerate() {
                    if ui.button(preset.name).clicked() {
                        theme::start_accent_transition(i);
                        theme::set_accent(i);
                    }
                }
            });
        });
    }
}
