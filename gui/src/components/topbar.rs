// 顶部栏：当前分区标题 + 深浅色切换。

use crate::app::DacApp;
use crate::theme::{self, colors};
use egui::FontId;

pub fn render(app: &mut DacApp, ctx: &egui::Context) {
    egui::TopBottomPanel::top("topbar")
        .min_height(56.0)
        .max_height(56.0)
        .resizable(false)
        .show_separator_line(false)
        .frame(
            egui::Frame::new()
                .fill(colors::bg_white())
                .inner_margin(egui::Margin::ZERO)
                .stroke(egui::Stroke::NONE),
        )
        .show(ctx, |ui| {
            let avail_h = ui.available_height();
            let top_pad = ((avail_h - 40.0) / 2.0).max(0.0);
            ui.add_space(top_pad);

            ui.horizontal(|ui| {
                ui.add_space(32.0);

                ui.label(
                    egui::RichText::new(app.current_section.title())
                        .font(FontId::new(20.0, egui::FontFamily::Proportional))
                        .color(colors::text_primary()),
                );

                ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                    ui.add_space(24.0);

                    // 深浅色切换（手动切换后脱离 System 跟随）
                    let icon = if theme::is_dark() { "☀️" } else { "🌙" };
                    let resp = ui
                        .add(
                            egui::Button::new(egui::RichText::new(icon).size(16.0))
                                .fill(colors::bg_card())
                                .stroke(egui::Stroke::NONE),
                        )
                        .on_hover_text(if theme::is_dark() {
                            "切换到浅色"
                        } else {
                            "切换到深色"
                        });
                    if resp.clicked() {
                        // 走 set_theme_mode 统一处理过渡与偏好持久化
                        let target = !theme::is_dark();
                        app.set_theme_mode(if target {
                            theme::ThemeMode::Dark
                        } else {
                            theme::ThemeMode::Light
                        });
                    }
                });
            });
        });
}
