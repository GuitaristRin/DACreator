// 数据页：历史记录浏览与赛道筛选（M3 接入引擎后填充）。

use crate::app::DacApp;

pub fn render(_app: &mut DacApp, ui: &mut egui::Ui) {
    ui.heading("数据");
    ui.add_space(8.0);
    ui.label(
        egui::RichText::new("历史成绩记录与进步追踪。")
            .color(crate::theme::colors::text_secondary()),
    );
}
