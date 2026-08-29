// 可折叠侧边栏：SpringAnim 驱动宽度 54↔200px，indicator 弹簧驱动左侧高亮滑块。
// 移植自 PezMax-One Metro 风格。

use crate::app::{DacApp, Section};
use crate::theme::{self, colors};
use egui::{pos2, Color32, CornerRadius, Frame, Rect, Sense};
use sokuou::map_range_clamped;

pub fn render(app: &mut DacApp, ctx: &egui::Context) {
    let anim_val = app.sidebar_anim.value().clamp(0.0, 1.0);
    let width = map_range_clamped(anim_val, 54.0, 200.0) as f32;
    // 导航标签渐入渐出 + 滑动（0.3→0.7 之间过渡）
    let label_alpha = ((anim_val - 0.3) / 0.4).clamp(0.0, 1.0) as f32;
    let label_offset = (1.0 - anim_val) as f32 * 6.0;

    let sidebar_fg = if theme::is_dark() {
        Color32::WHITE
    } else {
        Color32::from_rgb(0x3A, 0x30, 0x28)
    };
    let sidebar_fg_muted = if theme::is_dark() {
        Color32::from_gray(180)
    } else {
        Color32::from_rgb(0x70, 0x65, 0x55)
    };

    egui::SidePanel::left("sidebar")
        .resizable(false)
        .min_width(width)
        .max_width(width)
        .show_separator_line(false)
        .frame(
            egui::Frame::new()
                .fill(colors::bg_sidebar())
                .inner_margin(egui::Margin::ZERO)
                .stroke(egui::Stroke::NONE),
        )
        .show(ctx, |ui| {
            ui.set_min_width(width);
            ui.add_space(12.0);

            // ☰ 汉堡按钮（Label + Sense::click，避免 button_padding 溢出）
            ui.horizontal(|ui| {
                ui.add_space(12.0);
                let resp = ui
                    .label(
                        egui::RichText::new("☰")
                            .font(egui::FontId::new(22.0, egui::FontFamily::Proportional))
                            .color(sidebar_fg),
                    )
                    .interact(Sense::click())
                    .on_hover_cursor(egui::CursorIcon::PointingHand);
                if resp.clicked() {
                    app.sidebar_open = !app.sidebar_open;
                    app.sidebar_anim
                        .set_target(if app.sidebar_open { 1.0 } else { 0.0 });
                }
            });

            ui.add_space(24.0);

            // ── 导航项：收集 rect 供滑块插值 ──────────────────────
            let mut item_rects: [Option<Rect>; 4] = [None; 4];

            for (i, section) in Section::ALL.iter().enumerate() {
                let is_active = app.current_section == *section;

                let resp = Frame::new()
                    .fill(if is_active {
                        colors::bg_selected()
                    } else {
                        Color32::TRANSPARENT
                    })
                    .corner_radius(CornerRadius::same(0))
                    .show(ui, |ui| {
                        ui.set_min_width(width);
                        ui.add_space(8.0);
                        ui.horizontal(|ui| {
                            ui.add_space(3.0); // 滑块指示器宽度
                            ui.add_space(12.0);
                            ui.label(
                                egui::RichText::new(section.icon())
                                    .font(egui::FontId::new(22.0, egui::FontFamily::Proportional))
                                    .color(sidebar_fg),
                            );
                            if label_alpha > 0.0 {
                                ui.add_space(12.0 + label_offset);
                                ui.set_opacity(label_alpha);
                                ui.label(
                                    egui::RichText::new(section.title())
                                        .font(egui::FontId::new(
                                            16.0,
                                            egui::FontFamily::Proportional,
                                        ))
                                        .color(if is_active {
                                            sidebar_fg
                                        } else {
                                            sidebar_fg_muted
                                        }),
                                );
                            }
                        });
                        ui.add_space(8.0);
                    })
                    .response
                    .interact(Sense::click())
                    .on_hover_cursor(egui::CursorIcon::PointingHand);

                let clicked = resp.clicked();
                item_rects[i] = Some(resp.rect);

                // 折叠态用 tooltip 提示功能名
                if label_alpha < 0.5 {
                    resp.on_hover_text(section.title());
                }

                if clicked && !is_active {
                    app.navigate(*section);
                }
            }

            // ── 滑块指示器（弹簧插值 y）───────────────────────────
            let idx_f = app.sidebar_indicator_anim.value(); // 0.0 – 3.0
            let lo = idx_f.floor() as usize;
            let hi = (idx_f.ceil() as usize).min(3);
            let t = idx_f.fract() as f32;

            if let (Some(r_lo), Some(r_hi)) = (item_rects[lo], item_rects[hi]) {
                let y_top = egui::lerp(r_lo.top()..=r_hi.top(), t);
                let y_bot = egui::lerp(r_lo.bottom()..=r_hi.bottom(), t);
                let bar =
                    Rect::from_min_max(pos2(r_lo.left(), y_top), pos2(r_lo.left() + 3.0, y_bot));
                ui.painter()
                    .rect_filled(bar, CornerRadius::ZERO, colors::primary());
            }
        });
}
