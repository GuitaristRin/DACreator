// 设置页：玩家信息配置（经引擎 config show/set/import）+ 外观偏好（本地持久化）。

use std::sync::mpsc::Receiver;

use serde::{Deserialize, Serialize};

use crate::app::DacApp;
use crate::theme::{self, colors};

/// 引擎配置（`dac config show --json` 输出）。
#[derive(Deserialize, Debug, Default, Clone)]
pub struct EngineConfig {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub region: String,
    #[serde(default)]
    pub city: String,
    #[serde(default)]
    pub store: String,
    #[serde(default)]
    pub team: String,
    #[serde(default)]
    pub season: i32,
    #[serde(default)]
    pub round: i32,
}

/// GUI 外观偏好（本地持久化，独立于引擎配置）。
#[derive(Serialize, Deserialize, Debug, Clone, Copy, Default)]
pub struct UiPrefs {
    /// 0=System 1=Light 2=Dark
    pub theme_mode: u8,
    pub accent_idx: usize,
}

pub fn load_ui_prefs() -> UiPrefs {
    confy::load("DACreator", None).unwrap_or_default()
}

pub fn store_ui_prefs(prefs: UiPrefs) {
    if let Err(e) = confy::store("DACreator", None, &prefs) {
        eprintln!("⚠️ 保存外观偏好失败：{e}");
    }
}

/// 设置页状态。
#[derive(Default)]
pub struct SettingsState {
    pub loaded_once: bool,
    pub loading: bool,
    pub load_rx: Option<Receiver<std::io::Result<String>>>,
    pub form: EngineConfig,
    pub load_error: Option<String>,
    /// 保存任务通道与结果提示
    pub save_rx: Option<Receiver<std::io::Result<String>>>,
    pub save_status: Option<(bool, String)>, // (成功?, 文本)
}

impl SettingsState {
    fn request_load(&mut self, exe: &std::path::Path) {
        if self.loading {
            return;
        }
        self.load_rx = Some(crate::engine::query_engine(
            exe,
            &["config".to_owned(), "show".to_owned(), "--json".to_owned()],
        ));
        self.loading = true;
    }

    fn poll(&mut self, ctx: &egui::Context) {
        // 加载
        if let Some(rx) = &self.load_rx {
            match rx.try_recv() {
                Ok(Ok(stdout)) => {
                    match serde_json::from_str::<EngineConfig>(&stdout) {
                        Ok(cfg) => self.form = cfg,
                        Err(e) => self.load_error = Some(format!("解析配置失败：{e}")),
                    }
                    self.loading = false;
                    self.loaded_once = true;
                    self.load_rx = None;
                    ctx.request_repaint();
                }
                Ok(Err(e)) => {
                    self.load_error = Some(e.to_string());
                    self.loading = false;
                    self.loaded_once = true;
                    self.load_rx = None;
                    ctx.request_repaint();
                }
                Err(std::sync::mpsc::TryRecvError::Empty) => {}
                Err(std::sync::mpsc::TryRecvError::Disconnected) => {
                    self.load_error = Some("引擎查询通道中断".to_owned());
                    self.loading = false;
                    self.load_rx = None;
                }
            }
        }
        // 保存 / 导入
        if let Some(rx) = &self.save_rx {
            match rx.try_recv() {
                Ok(Ok(_)) => {
                    self.save_rx = None;
                    self.save_status = Some((true, "✅ 设置已保存".to_owned()));
                    ctx.request_repaint();
                }
                Ok(Err(e)) => {
                    self.save_rx = None;
                    self.save_status = Some((false, format!("❌ 保存失败：{e}")));
                    ctx.request_repaint();
                }
                Err(std::sync::mpsc::TryRecvError::Empty) => {}
                Err(std::sync::mpsc::TryRecvError::Disconnected) => {
                    self.save_rx = None;
                    self.save_status = Some((false, "❌ 保存通道中断".to_owned()));
                }
            }
        }
    }
}

pub fn render(app: &mut DacApp, ui: &mut egui::Ui) {
    ui.heading("设置");
    ui.add_space(8.0);

    // 首次进入加载引擎配置
    if !app.settings.loaded_once && !app.settings.loading {
        if let Some(exe) = app.engine_exe.clone() {
            app.settings.request_load(&exe);
        } else {
            app.settings.load_error = Some("未找到引擎程序 dac.exe".to_owned());
            app.settings.loaded_once = true;
        }
    }
    app.settings.poll(ui.ctx());

    // ── 玩家信息 ───────────────────────────────────────────────
    egui::Frame::new()
        .fill(colors::bg_card())
        .inner_margin(egui::Margin::same(16))
        .show(ui, |ui| {
            ui.set_width(ui.available_width());
            ui.label(egui::RichText::new("玩家信息").size(16.0));
            if app.settings.loading {
                ui.add(egui::Spinner::new().size(18.0));
                return;
            }
            if let Some(err) = &app.settings.load_error {
                ui.label(egui::RichText::new(format!("❌ {err}")).color(colors::error()));
                return;
            }

            let form = &mut app.settings.form;
            let w = (ui.available_width() - 220.0).max(200.0);
            field(ui, "👤 用户名（ArcadeZone ID）", &mut form.id, w);
            field(ui, "🗺️ 地区", &mut form.region, w);
            field(ui, "🏙️ 城市", &mut form.city, w);
            field(ui, "🏪 店铺", &mut form.store, w);
            field(ui, "🚗 车队", &mut form.team, w);
            ui.horizontal(|ui| {
                ui.label("📅 赛季：");
                ui.add(egui::DragValue::new(&mut form.season).range(1..=10));
                ui.add_space(24.0);
                ui.label("🔄 回合：");
                ui.add(egui::DragValue::new(&mut form.round).range(1..=10));
            });

            ui.add_space(12.0);
            ui.horizontal(|ui| {
                let exe = app.engine_exe.clone();
                let saving = app.settings.save_rx.is_some();
                let save_btn = egui::Button::new(egui::RichText::new("💾 保存设置").size(15.0))
                    .fill(colors::primary())
                    .stroke(egui::Stroke::NONE)
                    .min_size(egui::vec2(150.0, 40.0));
                if ui.add_enabled(exe.is_some() && !saving, save_btn).clicked() {
                    save_config(app);
                }
                if ui
                    .add_enabled(
                        exe.is_some() && !saving,
                        egui::Button::new("从旧版 Player_ID.dat 导入…"),
                    )
                    .clicked()
                {
                    if let Some(dat) = rfd::FileDialog::new()
                        .add_filter("旧版配置", &["dat"])
                        .add_filter("所有文件", &["*"])
                        .pick_file()
                    {
                        import_legacy(app, &dat.display().to_string());
                    }
                }
                if saving {
                    ui.add(egui::Spinner::new().size(16.0));
                }
                if let Some((ok, msg)) = &app.settings.save_status {
                    ui.label(egui::RichText::new(msg).color(if *ok {
                        colors::success()
                    } else {
                        colors::error()
                    }));
                }
            });
        });

    ui.add_space(16.0);

    // ── 外观 ───────────────────────────────────────────────────
    egui::Frame::new()
        .fill(colors::bg_card())
        .inner_margin(egui::Margin::same(16))
        .show(ui, |ui| {
            ui.set_width(ui.available_width());
            ui.label(egui::RichText::new("外观").size(16.0));
            ui.add_space(4.0);

            ui.horizontal(|ui| {
                ui.label("模式：");
                for (mode, label) in [
                    (theme::ThemeMode::System, "跟随系统"),
                    (theme::ThemeMode::Light, "浅色"),
                    (theme::ThemeMode::Dark, "深色"),
                ] {
                    if ui.radio(app.theme_mode == mode, label).clicked() {
                        app.set_theme_mode(mode);
                    }
                }
            });

            ui.horizontal(|ui| {
                ui.label("强调色：");
                for (i, preset) in theme::ACCENT_PRESETS.iter().enumerate() {
                    let selected = theme::accent_idx() == i;
                    let swatch = egui::Button::new(egui::RichText::new("　").size(14.0))
                        .fill(egui::Color32::from_rgb(preset.r, preset.g, preset.b))
                        .stroke(if selected {
                            egui::Stroke::new(2.0, colors::text_primary())
                        } else {
                            egui::Stroke::NONE
                        });
                    if ui.add(swatch).on_hover_text(preset.name).clicked() {
                        app.set_accent(i);
                    }
                }
            });
        });
}

fn field(ui: &mut egui::Ui, label: &str, value: &mut String, width: f32) {
    ui.horizontal(|ui| {
        ui.label(label);
        ui.add_sized([width, 28.0], egui::TextEdit::singleline(value));
    });
}

fn save_config(app: &mut DacApp) {
    let Some(exe) = app.engine_exe.clone() else {
        return;
    };
    let f = &app.settings.form;
    let args = vec![
        "config".to_owned(),
        "set".to_owned(),
        "--id".to_owned(),
        f.id.clone(),
        "--region".to_owned(),
        f.region.clone(),
        "--city".to_owned(),
        f.city.clone(),
        "--store".to_owned(),
        f.store.clone(),
        "--team".to_owned(),
        f.team.clone(),
        "--season".to_owned(),
        f.season.to_string(),
        "--round".to_owned(),
        f.round.to_string(),
    ];
    app.settings.save_rx = Some(crate::engine::query_engine(&exe, &args));
    app.settings.save_status = None;
}

fn import_legacy(app: &mut DacApp, dat_path: &str) {
    let Some(exe) = app.engine_exe.clone() else {
        return;
    };
    let args = vec![
        "config".to_owned(),
        "import".to_owned(),
        dat_path.to_owned(),
    ];
    app.settings.save_rx = Some(crate::engine::query_engine(&exe, &args));
    // 导入完成后重新加载表单
    app.settings.loaded_once = false;
    app.settings.save_status = None;
}
