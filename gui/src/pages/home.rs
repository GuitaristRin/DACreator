// 主页：任务发起、实时进度与日志控制台。

use crate::app::DacApp;
use crate::theme::colors;
use egui::{FontId, RichText};

/// 主页可选任务模式。
#[derive(Clone, Copy, PartialEq, Eq, Debug, Default)]
pub enum HomeMode {
    #[default]
    Crawl,
    LocalCsv,
    Card,
    RecordCard,
}

impl HomeMode {
    fn label(self) -> &'static str {
        match self {
            HomeMode::Crawl => "🌐 成绩爬取（全部赛道，含全国排名）",
            HomeMode::LocalCsv => "📁 本地 CSV 生成表格",
            HomeMode::Card => "成绩卡（简报：Top5 + 回合/名声/车队）",
            HomeMode::RecordCard => "记录卡（玩家 ID + 车队 + 等级统计 + 精选成绩）",
        }
    }
}

/// 一条日志行（时间为本次任务开始后的相对秒数）。
pub struct LogLine {
    pub elapsed_secs: f64,
    pub level: String,
    pub msg: String,
}

impl LogLine {
    fn level_color(&self) -> egui::Color32 {
        match self.level.as_str() {
            "success" => colors::success(),
            "warning" => colors::warning(),
            "error" => colors::error(),
            _ => colors::text_primary(),
        }
    }

    fn icon(&self) -> &'static str {
        match self.level.as_str() {
            "success" => "✅",
            "warning" => "⚠️",
            "error" => "❌",
            _ => "📌",
        }
    }
}

/// 最近一次任务结果。
pub struct TaskResult {
    pub csv_path: String,
    pub png_path: Option<String>,
    pub records: u64,
    pub elapsed_ms: u64,
}

pub fn render(app: &mut DacApp, ui: &mut egui::Ui) {
    ui.heading("DACreator");
    ui.label(
        RichText::new("爬取 ArcadeZone 计时赛成绩，生成可视化表格并记录历史。")
            .color(colors::text_secondary()),
    );
    ui.add_space(16.0);

    // ── 模式选择 ────────────────────────────────────────────────
    egui::Frame::new()
        .fill(colors::bg_card())
        .inner_margin(egui::Margin::same(16))
        .show(ui, |ui| {
            ui.set_width(ui.available_width());
            ui.label(RichText::new("选择模式").size(16.0));
            ui.add_space(4.0);
            for mode in [
                HomeMode::Crawl,
                HomeMode::LocalCsv,
                HomeMode::Card,
                HomeMode::RecordCard,
            ] {
                ui.radio_value(&mut app.home_mode, mode, mode.label());
            }

            if app.home_mode == HomeMode::LocalCsv {
                ui.add_space(8.0);
                ui.horizontal(|ui| {
                    ui.label("CSV 文件：");
                    let mut buf = app.csv_path.clone().unwrap_or_default();
                    let resp = ui.add_sized(
                        [ui.available_width() - 110.0, 28.0],
                        egui::TextEdit::singleline(&mut buf),
                    );
                    if resp.changed() {
                        app.csv_path = if buf.is_empty() { None } else { Some(buf) };
                    }
                    if ui.button("浏览 CSV…").clicked() {
                        if let Some(path) = rfd::FileDialog::new()
                            .add_filter("CSV 文件", &["csv"])
                            .add_filter("所有文件", &["*"])
                            .pick_file()
                        {
                            app.csv_path = Some(path.display().to_string());
                        }
                    }
                });
            }

            ui.add_space(4.0);
            ui.horizontal(|ui| {
                ui.label(RichText::new("图片输出目录（可选）：").color(colors::text_secondary()));
                let mut buf = app.out_dir.clone().unwrap_or_default();
                let resp = ui.add_sized([280.0, 28.0], egui::TextEdit::singleline(&mut buf));
                if resp.changed() {
                    app.out_dir = if buf.is_empty() { None } else { Some(buf) };
                }
                if ui.button("浏览…").clicked() {
                    if let Some(dir) = rfd::FileDialog::new().pick_folder() {
                        app.out_dir = Some(dir.display().to_string());
                    }
                }
            });
        });

    ui.add_space(16.0);

    // ── 进度与日志 ──────────────────────────────────────────────
    egui::Frame::new()
        .fill(colors::bg_card())
        .inner_margin(egui::Margin::same(16))
        .show(ui, |ui| {
            ui.set_width(ui.available_width());

            if let Some((stage, pct, detail)) = &app.progress {
                ui.horizontal(|ui| {
                    ui.label(RichText::new(format!("【{stage}】")).color(colors::primary()));
                    ui.label(RichText::new(detail).color(colors::text_secondary()));
                });
                ui.add(
                    egui::ProgressBar::new((*pct as f32 / 100.0).clamp(0.0, 1.0))
                        .desired_height(8.0)
                        .fill(colors::primary()),
                );
            } else if app.task_running {
                ui.label(RichText::new("任务进行中…").color(colors::text_secondary()));
            } else {
                ui.label(RichText::new("就绪").color(colors::text_secondary()));
            }

            ui.add_space(8.0);
            let log_height = (ui.available_height() - 70.0).max(160.0);
            egui::ScrollArea::vertical()
                .auto_shrink([false, false])
                .stick_to_bottom(true)
                .max_height(log_height)
                .show(ui, |ui| {
                    if app.logs.is_empty() {
                        ui.label(RichText::new("日志将在此显示。").color(colors::text_secondary()));
                    }
                    for line in &app.logs {
                        ui.horizontal(|ui| {
                            ui.label(
                                RichText::new(format!("[{:8.1}s]", line.elapsed_secs))
                                    .font(FontId::monospace(12.0))
                                    .color(colors::text_secondary()),
                            );
                            ui.label(
                                RichText::new(line.icon())
                                    .font(FontId::monospace(12.0))
                                    .color(line.level_color()),
                            );
                            ui.label(
                                RichText::new(&line.msg)
                                    .font(FontId::monospace(12.0))
                                    .color(line.level_color()),
                            );
                        });
                    }
                });

            ui.add_space(12.0);
            ui.horizontal(|ui| {
                let start_label = if app.task_running {
                    "任务进行中…"
                } else {
                    "🚀 开始"
                };
                let start_btn = egui::Button::new(RichText::new(start_label).size(16.0))
                    .fill(if app.task_running {
                        colors::bg_input()
                    } else {
                        colors::primary()
                    })
                    .stroke(egui::Stroke::NONE)
                    .min_size(egui::vec2(160.0, 44.0));
                if ui.add_enabled(!app.task_running, start_btn).clicked() {
                    app.start_home_task();
                }

                let stop_btn = egui::Button::new(RichText::new("⏹ 停止").size(16.0))
                    .fill(colors::bg_input())
                    .stroke(egui::Stroke::NONE)
                    .min_size(egui::vec2(120.0, 44.0));
                if ui.add_enabled(app.task_running, stop_btn).clicked() {
                    app.stop_home_task();
                }

                if let Some(result) = &app.last_result {
                    ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                        // 成绩卡只有图片；爬取/本地 CSV 未指定输出目录时回退到原始 CSV 所在目录
                        let csv = (!result.csv_path.is_empty()).then_some(result.csv_path.as_str());
                        let open_path = result.png_path.as_deref().or(csv);
                        if let Some(path) = open_path {
                            let label = if result.png_path.is_some() {
                                "打开图片目录"
                            } else {
                                "打开数据目录"
                            };
                            if ui.button(label).clicked() {
                                open_containing_dir(path);
                            }
                        }
                        ui.label(
                            RichText::new(format!(
                                "最近一次：{} 条记录，耗时 {:.1}s",
                                result.records,
                                result.elapsed_ms as f64 / 1000.0
                            ))
                            .color(colors::success()),
                        );
                    });
                }
            });

            if let Some(err) = &app.last_error {
                ui.add_space(8.0);
                ui.label(RichText::new(format!("❌ {err}")).color(colors::error()));
            }
        });
}

fn open_containing_dir(path: &str) {
    let p = std::path::Path::new(path);
    let dir = p.parent().unwrap_or(p);
    #[cfg(windows)]
    let _ = std::process::Command::new("explorer").arg(dir).spawn();
    #[cfg(target_os = "macos")]
    let _ = std::process::Command::new("open").arg(dir).spawn();
    #[cfg(all(unix, not(target_os = "macos")))]
    let _ = std::process::Command::new("xdg-open").arg(dir).spawn();
}
