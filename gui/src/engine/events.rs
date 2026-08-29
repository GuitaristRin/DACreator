// 引擎事件（internal/events 的 JSON 契约镜像）。字段全部可缺省，解析保持宽容。

use serde::Deserialize;

#[derive(Deserialize, Debug, Clone)]
pub struct EngineEvent {
    #[serde(rename = "type")]
    pub kind: String,
    #[serde(default)]
    pub stage: String,
    #[serde(default)]
    pub pct: i32,
    #[serde(default)]
    pub detail: String,
    #[serde(default)]
    pub level: String,
    #[serde(default)]
    pub msg: String,
    #[serde(default)]
    pub code: String,
    #[serde(default)]
    pub csv_path: String,
    #[serde(default)]
    pub png_path: String,
    #[serde(default)]
    pub records: u64,
    #[serde(default)]
    pub elapsed_ms: u64,
}
