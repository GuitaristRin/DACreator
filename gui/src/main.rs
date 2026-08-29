//! DACreator GUI（egui/eframe）。
//! v3 架构：GUI 只负责界面与动画，数据操作全部通过引擎子进程（cmd/dac）完成。

mod app;
mod components;
mod engine;
mod pages;
mod theme;

use eframe::egui;

fn main() -> eframe::Result {
    // 二合一：带参数时作为 CLI 透传给引擎（DACreator.exe crawl …）
    if std::env::args().count() > 1 {
        let args: Vec<String> = std::env::args().skip(1).collect();
        engine::ensure_engine_extracted();
        let Some(exe) = engine::resolve_engine_exe() else {
            eprintln!("未找到引擎程序 dac.exe");
            std::process::exit(2);
        };
        let status = std::process::Command::new(exe)
            .args(&args)
            .status()
            .expect("启动引擎失败");
        std::process::exit(status.code().unwrap_or(1));
    }

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
