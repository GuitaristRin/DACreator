// 关于页：版本信息、开发者、依赖与更新检查（M3 接入引擎后填充）。

use crate::app::DacApp;

pub fn render(_app: &mut DacApp, ui: &mut egui::Ui) {
    ui.heading("关于");
    ui.add_space(8.0);
    ui.label(
        egui::RichText::new("DACreator v3 —— 为《头文字D 激斗》玩家打造的成绩工具。")
            .color(crate::theme::colors::text_secondary()),
    );
}
