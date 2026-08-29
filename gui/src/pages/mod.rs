// 页面路由：按当前分区渲染对应页面。页面为纯函数，只读写 DacApp。

pub mod about;
pub mod home;
pub mod records;
pub mod settings;

use crate::app::{DacApp, Section};

pub fn dispatch(app: &mut DacApp, ctx: &egui::Context) {
    egui::CentralPanel::default()
        .frame(
            egui::Frame::new()
                .fill(crate::theme::colors::bg_white())
                .inner_margin(egui::Margin::same(32)),
        )
        .show(ctx, |ui| match app.current_section {
            Section::Home => home::render(app, ui),
            Section::Records => records::render(app, ui),
            Section::Settings => settings::render(app, ui),
            Section::About => about::render(app, ui),
        });
}
