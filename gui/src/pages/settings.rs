// 设置页：玩家信息与外观配置（M3 接入引擎后填充）。

use crate::app::DacApp;

pub fn render(_app: &mut DacApp, ui: &mut egui::Ui) {
    ui.heading("设置");
    ui.add_space(8.0);

    ui.horizontal(|ui| {
        ui.label("外观：");
        for (i, preset) in crate::theme::ACCENT_PRESETS.iter().enumerate() {
            let (r, g, b) = (preset.r, preset.g, preset.b);
            let swatch = egui::Button::new(egui::RichText::new("　").size(14.0))
                .fill(egui::Color32::from_rgb(r, g, b))
                .stroke(if crate::theme::accent_idx() == i {
                    egui::Stroke::new(2.0, crate::theme::colors::text_primary())
                } else {
                    egui::Stroke::NONE
                });
            if ui.add(swatch).on_hover_text(preset.name).clicked() {
                crate::theme::start_accent_transition(i);
                crate::theme::set_accent(i);
            }
        }
    });
}
