//! DACreator GUI（egui/eframe）。
//! v3 架构：GUI 只负责界面与动画，数据操作全部通过引擎子进程（cmd/dac）完成。

use eframe::egui;

fn main() -> eframe::Result {
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default().with_inner_size([1200.0, 800.0]),
        ..Default::default()
    };
    eframe::run_native(
        "DACreator",
        options,
        Box::new(|_cc| Ok(Box::new(DacApp::new()))),
    )
}

/// 单一状态结构：所有应用状态挂在这里，页面渲染为纯函数（对齐 PezMax-One 模式）。
struct DacApp {
    // M2/M3 阶段填充：theme、sokuou 动画、EngineHandle、页面路由。
}

impl DacApp {
    fn new() -> Self {
        Self {}
    }
}

impl eframe::App for DacApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        egui::CentralPanel::default().show(ctx, |ui| {
            ui.heading("DACreator v3");
            ui.label("引擎与界面重构进行中…");
        });
    }
}
