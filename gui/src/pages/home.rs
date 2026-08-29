// 主页：任务发起与实时进度（M3 接入引擎后填充）。

use crate::app::DacApp;

pub fn render(_app: &mut DacApp, ui: &mut egui::Ui) {
    ui.heading("DACreator");
    ui.add_space(8.0);
    ui.label(
        egui::RichText::new("爬取 ArcadeZone 计时赛成绩，生成可视化表格。")
            .color(crate::theme::colors::text_secondary()),
    );
}
