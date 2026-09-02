// 数据页：历史记录浏览与赛道筛选（数据来自引擎 `dac history --json`）。

use std::sync::mpsc::Receiver;

use serde::Deserialize;

use crate::app::DacApp;
use crate::theme::colors;

/// 与引擎 `dac history --json` 输出对应的一条记录。
#[derive(Deserialize, Clone, Debug)]
pub struct HistoryRecord {
    #[serde(default)]
    pub course: String,
    #[serde(default)]
    pub direction: String,
    #[serde(default)]
    pub time_str: String,
    #[serde(default)]
    pub rank: String,
    #[serde(default)]
    pub car: String,
    #[serde(default)]
    pub national_rank: String,
    #[serde(default)]
    pub record_date: String,
    #[serde(default)]
    pub created_at: String,
}

#[derive(Deserialize)]
struct HistoryResp {
    #[serde(default)]
    records: Vec<HistoryRecord>,
}

/// 历史数据加载状态（对齐 PezMax-One 的 AsyncData 模式）。
#[derive(Default)]
pub struct HistoryState {
    pub loading: bool,
    pub loaded_once: bool,
    pub rx: Option<Receiver<std::io::Result<String>>>,
    pub all: Vec<HistoryRecord>,
    pub error: Option<String>,
    /// 赛道筛选，空串 = 全部。
    pub course_filter: String,
}

impl HistoryState {
    /// 触发一次加载（幂等：加载中不重复触发）。
    pub fn request_load(&mut self, exe: &std::path::Path) {
        if self.loading {
            return;
        }
        let args = vec![
            "history".to_owned(),
            "-n".to_owned(),
            "500".to_owned(),
            "--json".to_owned(),
        ];
        self.rx = Some(crate::engine::query_engine(exe, &args));
        self.loading = true;
        self.error = None;
    }

    /// 非阻塞轮询加载结果。
    pub fn poll(&mut self, ctx: &egui::Context) {
        let Some(rx) = &self.rx else { return };
        let result = match rx.try_recv() {
            Ok(r) => Some(r),
            Err(std::sync::mpsc::TryRecvError::Empty) => return,
            Err(std::sync::mpsc::TryRecvError::Disconnected) => {
                Some(Err(std::io::Error::other("引擎查询通道中断")))
            }
        };
        self.rx = None;
        self.loading = false;

        match result {
            Some(Ok(stdout)) => match serde_json::from_str::<HistoryResp>(&stdout) {
                Ok(resp) => {
                    self.all = resp.records;
                    self.loaded_once = true;
                }
                Err(e) => self.error = Some(format!("解析历史数据失败：{e}")),
            },
            Some(Err(e)) => self.error = Some(e.to_string()),
            None => self.error = Some("引擎查询通道中断".to_owned()),
        }
        ctx.request_repaint();
    }

    /// 当前筛选下应显示的记录。
    pub fn filtered(&self) -> Vec<&HistoryRecord> {
        self.all
            .iter()
            .filter(|r| self.course_filter.is_empty() || r.course == self.course_filter)
            .collect()
    }

    /// 全部赛道名（去重升序）。
    fn courses(&self) -> Vec<String> {
        let mut courses: Vec<String> = self.all.iter().map(|r| r.course.clone()).collect();
        courses.sort();
        courses.dedup();
        courses
    }
}

pub fn render(app: &mut DacApp, ui: &mut egui::Ui) {
    ui.heading("数据");
    ui.add_space(8.0);

    // 首次进入自动加载
    if !app.history.loaded_once && !app.history.loading {
        if let Some(exe) = app.engine_exe.clone() {
            app.history.request_load(&exe);
        } else {
            app.history.error = Some("未找到引擎程序 dac.exe".to_owned());
            app.history.loaded_once = true;
        }
    }
    app.history.poll(ui.ctx());

    // ── 工具行：筛选 + 刷新 ────────────────────────────────────
    ui.horizontal(|ui| {
        ui.label("赛道：");
        let courses = app.history.courses();
        let current = app.history.course_filter.clone();
        egui::ComboBox::from_id_salt("course_filter")
            .selected_text(if current.is_empty() {
                "全部".to_owned()
            } else {
                current.clone()
            })
            .width(160.0)
            .show_ui(ui, |ui| {
                ui.selectable_value(&mut app.history.course_filter, String::new(), "全部");
                for c in &courses {
                    ui.selectable_value(&mut app.history.course_filter, c.clone(), c);
                }
            });

        ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
            let exe = app.engine_exe.clone();
            if ui
                .add_enabled(
                    exe.is_some() && !app.history.loading,
                    egui::Button::new("🔄 刷新"),
                )
                .clicked()
            {
                if let Some(exe) = exe {
                    app.history.loaded_once = false;
                    app.history.request_load(&exe);
                }
            }
            if app.history.loading {
                ui.add(egui::Spinner::new().size(18.0));
            }
        });
    });
    ui.add_space(4.0);

    // ── 状态展示 ───────────────────────────────────────────────
    if let Some(err) = &app.history.error {
        ui.label(egui::RichText::new(format!("❌ {err}")).color(colors::error()));
        return;
    }
    if app.history.loaded_once && app.history.all.is_empty() {
        ui.label(
            egui::RichText::new("暂无历史记录。先在主页完成一次任务吧。")
                .color(colors::text_secondary()),
        );
        return;
    }

    // ── 记录表格 ───────────────────────────────────────────────
    let rows = app.history.filtered();
    egui::ScrollArea::vertical()
        .auto_shrink([false, false])
        .show(ui, |ui| {
            use egui::TextWrapMode;
            ui.set_min_width(ui.available_width());
            egui::Grid::new("history_grid")
                .striped(true)
                .num_columns(8)
                .min_col_width(60.0)
                .show(ui, |ui| {
                    let header = |ui: &mut egui::Ui, text: &str| {
                        ui.label(
                            egui::RichText::new(text)
                                .strong()
                                .color(colors::text_secondary()),
                        );
                    };
                    header(ui, "记录日期");
                    header(ui, "赛道");
                    header(ui, "方向");
                    header(ui, "时间");
                    header(ui, "等级");
                    header(ui, "车型");
                    header(ui, "排名");
                    header(ui, "录入时间");
                    ui.end_row();

                    for r in &rows {
                        ui.label(&r.record_date);
                        ui.label(&r.course);
                        ui.label(&r.direction);
                        ui.label(
                            egui::RichText::new(&r.time_str)
                                .font(egui::FontId::monospace(13.0))
                                .color(colors::primary()),
                        );
                        ui.label(&r.rank);
                        ui.add(egui::Label::new(&r.car).wrap_mode(TextWrapMode::Truncate));
                        let national = if r.national_rank.is_empty() {
                            "—".to_owned()
                        } else {
                            format!("{}位", r.national_rank)
                        };
                        ui.label(national);
                        ui.label(
                            egui::RichText::new(format_created(&r.created_at))
                                .color(colors::text_secondary()),
                        );
                        ui.end_row();
                    }
                });
        });
}

/// RFC3339 → "YYYY-MM-DD HH:MM"（失败时原样返回）。
fn format_created(s: &str) -> String {
    let s = s.replace('T', " ");
    match (s.find(' '), s.rfind(':')) {
        (Some(sp), Some(colon)) if colon > sp => s[..colon].to_owned(),
        _ => s,
    }
}
