// 构建脚本：DAC_EMBED_ENGINE 指向待内嵌的引擎二进制时启用内嵌形态（单文件二合一）。
fn main() {
    println!("cargo:rerun-if-env-changed=DAC_EMBED_ENGINE");
    println!("cargo:rerun-if-env-changed=DAC_EMBED_VERSION");
    println!("cargo:rerun-if-changed=build.rs");

    let Ok(path) = std::env::var("DAC_EMBED_ENGINE") else {
        return;
    };
    if path.is_empty() || !std::path::Path::new(&path).is_file() {
        return;
    }
    let version = std::env::var("DAC_EMBED_VERSION").unwrap_or_default();
    let abs = std::fs::canonicalize(&path).expect("解析内嵌引擎路径");
    println!("cargo:rustc-env=DAC_EMBED_ENGINE={}", abs.display());
    println!("cargo:rustc-env=DAC_EMBED_VERSION={version}");
    println!("cargo:rustc-cfg=has_embed_engine");
}
