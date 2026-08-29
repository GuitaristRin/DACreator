//! DACreator GUI（egui/eframe）。
//! v3 架构：GUI 只负责界面与动画，数据操作全部通过引擎子进程（cmd/dac）完成。

mod app;
mod components;
mod engine;
mod pages;
mod theme;

use eframe::egui;

fn main() -> eframe::Result {
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default().with_inner_size([1200.0, 800.0]),
        ..Default::default()
    };
    eframe::run_native(
        "DACreator",
        options,
        Box::new(|cc| Ok(Box::new(app::DacApp::new(cc)))),
    )
}
