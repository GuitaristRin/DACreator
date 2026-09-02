// 关于页：版本信息、更新检查、开发者与免责声明。

use std::sync::mpsc::Receiver;

use serde::Deserialize;

use crate::app::DacApp;
use crate::theme::colors;

/// `dac update check --json` 输出。
#[derive(Deserialize, Debug, Default, Clone)]
pub struct UpdateInfo {
    #[serde(default)]
    pub current: String,
    #[serde(default)]
    pub latest: String,
    #[serde(default)]
    pub has_update: bool,
    #[serde(default)]
    pub notes: String,
    #[serde(default)]
    pub asset_name: String,
    #[serde(default)]
    pub asset_url: String,
}

#[derive(Default)]
pub struct AboutState {
    pub version: Option<String>,
    pub version_rx: Option<Receiver<std::io::Result<String>>>,
    pub update_checking: bool,
    pub update_rx: Option<Receiver<std::io::Result<String>>>,
    pub update_info: Option<UpdateInfo>,
    pub error: Option<String>,
}

impl AboutState {
    fn poll(&mut self, ctx: &egui::Context) {
        if let Some(rx) = &self.version_rx {
            match rx.try_recv() {
                Ok(Ok(stdout)) => {
                    #[derive(Deserialize)]
                    struct V {
                        #[serde(default)]
                        version: String,
                    }
                    self.version = serde_json::from_str::<V>(&stdout).ok().map(|v| v.version);
                    self.version_rx = None;
                    ctx.request_repaint();
                }
                Ok(Err(e)) => {
                    self.error = Some(format!("获取引擎版本失败：{e}"));
                    self.version_rx = None;
                    ctx.request_repaint();
                }
                Err(std::sync::mpsc::TryRecvError::Empty) => {}
                Err(std::sync::mpsc::TryRecvError::Disconnected) => {
                    self.version_rx = None;
                }
            }
        }
        if let Some(rx) = &self.update_rx {
            match rx.try_recv() {
                Ok(Ok(stdout)) => {
                    self.update_info = serde_json::from_str::<UpdateInfo>(&stdout).ok();
                    self.update_checking = false;
                    self.update_rx = None;
                    ctx.request_repaint();
                }
                Ok(Err(e)) => {
                    self.error = Some(e.to_string());
                    self.update_checking = false;
                    self.update_rx = None;
                    ctx.request_repaint();
                }
                Err(std::sync::mpsc::TryRecvError::Empty) => {}
                Err(std::sync::mpsc::TryRecvError::Disconnected) => {
                    self.update_checking = false;
                    self.update_rx = None;
                }
            }
        }
    }
}

pub fn render(app: &mut DacApp, ui: &mut egui::Ui) {
    ui.heading("关于");
    ui.add_space(8.0);
    app.about.poll(ui.ctx());

    // 首次进入查询引擎版本；引擎缺失时回退显示构建期内嵌的引擎版本
    if app.about.version.is_none() && app.about.version_rx.is_none() && !app.about.update_checking {
        if let Some(exe) = app.engine_exe.clone() {
            let args = vec!["version".to_owned(), "--json".to_owned()];
            app.about.version_rx = Some(crate::engine::query_engine(&exe, &args));
        } else if let Some(v) = crate::engine::embedded_version() {
            app.about.version = Some(v.to_owned());
        }
    }

    egui::Frame::new()
        .fill(colors::bg_card())
        .inner_margin(egui::Margin::same(16))
        .show(ui, |ui| {
            ui.set_width(ui.available_width());
            ui.label(egui::RichText::new("DACreator").size(24.0));
            ui.label(
                egui::RichText::new("为《头文字D 激斗》玩家打造的成绩工具：爬取 ArcadeZone 计时赛成绩、生成可视化表格与成绩卡、记录历史数据。")
                    .color(colors::text_secondary()),
            );
            ui.add_space(8.0);

            ui.horizontal(|ui| {
                ui.label("引擎版本：");
                match &app.about.version {
                    Some(v) => ui.label(egui::RichText::new(v).strong()),
                    None => ui.label(egui::RichText::new("获取中…").color(colors::text_secondary())),
                };
                if let Some(exe) = app.engine_exe.clone() {
                    let checking = app.about.update_checking;
                    let btn_label = if checking { "检查中…" } else { "🔍 检查更新" };
                    if ui.add_enabled(!checking, egui::Button::new(btn_label)).clicked() {
                        let args = vec!["update".to_owned(), "check".to_owned(), "--json".to_owned()];
                        app.about.update_rx = Some(crate::engine::query_engine(&exe, &args));
                        app.about.update_checking = true;
                        app.about.error = None;
                    }
                }
            });

            if let Some(info) = &app.about.update_info {
                ui.add_space(4.0);
                if info.has_update {
                    ui.label(
                        egui::RichText::new(format!(
                            "🎉 发现新版本 {}（当前 {}）",
                            info.latest, info.current
                        ))
                        .color(colors::success()),
                    );
                    if !info.notes.is_empty() {
                        egui::ScrollArea::vertical()
                            .max_height(120.0)
                            .show(ui, |ui| {
                                ui.label(egui::RichText::new(&info.notes).color(colors::text_secondary()));
                            });
                    }
                    ui.horizontal(|ui| {
                        if ui.button("前往 Releases 页面下载").clicked() {
                            open_url("https://github.com/GuitaristRin/DACreator/releases/latest");
                        }
                        if !info.asset_url.is_empty() {
                            ui.hyperlink_to(
                                format!("直接下载 {}", info.asset_name),
                                &info.asset_url,
                            );
                        }
                    });
                } else {
                    ui.label(
                        egui::RichText::new(format!("✅ 已是最新版本（{}）", info.current))
                            .color(colors::success()),
                    );
                }
            }
            if let Some(err) = &app.about.error {
                ui.label(egui::RichText::new(format!("❌ {err}")).color(colors::error()));
            }
        });

    ui.add_space(16.0);

    egui::Frame::new()
        .fill(colors::bg_card())
        .inner_margin(egui::Margin::same(16))
        .show(ui, |ui| {
            ui.set_width(ui.available_width());
            ui.label(egui::RichText::new("开发者").size(16.0));
            ui.label("核心开发：TakahashiRinta");
            ui.label("特别感谢：ArcadeZone 社区");
            ui.hyperlink_to(
                "GitHub：github.com/GuitaristRin/DACreator",
                "https://github.com/GuitaristRin/DACreator",
            );
            ui.add_space(8.0);
            ui.label(egui::RichText::new("技术栈").size(16.0));
            ui.label("引擎 Go · 界面 Rust/egui · 动画 Sokuou Engine · 字体 Noto Sans CJK");
            ui.add_space(8.0);
            ui.label(egui::RichText::new("免责声明").size(16.0));
            ui.label(
                egui::RichText::new("本项目遵循 MIT 开源协议，仅供学习交流，严禁商业用途。")
                    .color(colors::warning()),
            );
        });
}

fn open_url(url: &str) {
    #[cfg(windows)]
    let _ = std::process::Command::new("explorer").arg(url).spawn();
    #[cfg(target_os = "macos")]
    let _ = std::process::Command::new("open").arg(url).spawn();
    #[cfg(all(unix, not(target_os = "macos")))]
    let _ = std::process::Command::new("xdg-open").arg(url).spawn();
}
