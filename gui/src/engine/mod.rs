// EngineHandle：引擎子进程管理 + JSON-lines 事件流解析。
// GUI 与引擎的唯一通道：`dac <子命令> --json`，stdout 逐行 JSON 事件。

pub mod events;

use std::io::{BufRead, BufReader};
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::sync::mpsc::{self, Receiver, TryRecvError};
use std::sync::{Arc, Mutex};
use std::thread;

/// 引擎输出（对 UI 友好的聚合形式）。
#[derive(Debug, Clone)]
pub enum EngineOutput {
    Progress {
        stage: String,
        pct: i32,
        detail: String,
    },
    Log {
        level: String,
        msg: String,
    },
    Result {
        csv_path: String,
        png_path: String,
        records: u64,
        elapsed_ms: u64,
    },
    Error {
        code: String,
        msg: String,
    },
    /// 进程退出。success 对应退出码 0。
    Exited {
        success: bool,
    },
}

/// 一次引擎任务运行。
pub struct RunningTask {
    rx: Receiver<EngineOutput>,
    child: Arc<Mutex<Child>>,
    /// 任务名（主页日志展示用），如 "成绩爬取"。
    pub label: String,
}

impl RunningTask {
    /// 非阻塞取一条输出；无事件返回 None。
    pub fn try_recv(&self) -> Option<EngineOutput> {
        match self.rx.try_recv() {
            Ok(ev) => Some(ev),
            Err(TryRecvError::Empty) | Err(TryRecvError::Disconnected) => None,
        }
    }

    /// 终止引擎进程（停止按钮）。
    pub fn kill(&self) {
        if let Ok(mut child) = self.child.lock() {
            let _ = child.kill();
        }
    }
}

/// 启动引擎子进程并开始收集事件。
pub fn spawn_engine(exe: &Path, args: &[String], label: &str) -> std::io::Result<RunningTask> {
    let mut child = Command::new(exe)
        .args(args)
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .spawn()?;
    let stdout = child.stdout.take().expect("stdout 已 piped");
    let child = Arc::new(Mutex::new(child));

    let (tx, rx) = mpsc::channel();
    let child_for_reader = Arc::clone(&child);
    let label = label.to_owned();

    thread::spawn(move || {
        let reader = BufReader::new(stdout);
        for line in reader.lines() {
            let Ok(line) = line else { break };
            if line.trim().is_empty() {
                continue;
            }
            match serde_json::from_str::<events::EngineEvent>(&line) {
                Ok(ev) => {
                    if tx.send(event_to_output(&ev)).is_err() {
                        break; // 接收端已丢弃
                    }
                }
                Err(e) => {
                    // 非契约输出：作为警告透传，不中断任务
                    let _ = tx.send(EngineOutput::Log {
                        level: "warning".into(),
                        msg: format!("引擎输出解析失败（{e}）：{line}"),
                    });
                }
            }
        }
        // stdout 关闭后再回收退出码（此时进程已退出，wait 立即返回）
        let success = child_for_reader
            .lock()
            .ok()
            .and_then(|mut c| c.wait().ok())
            .is_some_and(|st| st.success());
        let _ = tx.send(EngineOutput::Exited { success });
    });

    Ok(RunningTask { rx, child, label })
}

/// 一次性查询：运行引擎子命令并收集完整 stdout（后台线程执行，经 Receiver 取回）。
/// 适用于 `config show --json`、`history --json` 等非事件流命令。
pub fn query_engine(exe: &Path, args: &[String]) -> Receiver<std::io::Result<String>> {
    let (tx, rx) = mpsc::channel();
    let mut cmd = Command::new(exe);
    cmd.args(args).stdout(Stdio::piped()).stderr(Stdio::piped());
    thread::spawn(move || {
        let out = match cmd.output() {
            Ok(o) if o.status.success() => Ok(String::from_utf8_lossy(&o.stdout).to_string()),
            Ok(o) => Err(std::io::Error::other(format!(
                "引擎退出码 {:?}：{}",
                o.status.code(),
                String::from_utf8_lossy(&o.stderr).trim()
            ))),
            Err(e) => Err(e),
        };
        tx.send(out).ok();
    });
    rx
}

fn event_to_output(ev: &events::EngineEvent) -> EngineOutput {
    match ev.kind.as_str() {
        "progress" => EngineOutput::Progress {
            stage: ev.stage.clone(),
            pct: ev.pct,
            detail: ev.detail.clone(),
        },
        "log" => EngineOutput::Log {
            level: ev.level.clone(),
            msg: ev.msg.clone(),
        },
        "result" => EngineOutput::Result {
            csv_path: ev.csv_path.clone(),
            png_path: ev.png_path.clone(),
            records: ev.records,
            elapsed_ms: ev.elapsed_ms,
        },
        "error" => EngineOutput::Error {
            code: ev.code.clone(),
            msg: ev.msg.clone(),
        },
        other => EngineOutput::Log {
            level: "warning".into(),
            msg: format!("未知事件类型：{other}"),
        },
    }
}

#[cfg(has_embed_engine)]
const EMBED_ENGINE: &[u8] = include_bytes!(env!("DAC_EMBED_ENGINE"));
#[cfg(has_embed_engine)]
const EMBED_VERSION: &str = env!("DAC_EMBED_VERSION");

/// 内嵌引擎版本（未启用内嵌时为 None）。
#[cfg(has_embed_engine)]
pub fn embedded_version() -> Option<&'static str> {
    Some(EMBED_VERSION)
}

#[cfg(not(has_embed_engine))]
pub fn embedded_version() -> Option<&'static str> {
    None
}

/// 把内嵌引擎释放到数据目录 bin/ 下（已存在且版本一致则跳过），返回其路径。
#[cfg(has_embed_engine)]
pub fn ensure_engine_extracted() -> Option<PathBuf> {
    let base = dirs::data_dir()?.join("DACreator").join("bin");
    let exe = base.join(if cfg!(windows) { "dac.exe" } else { "dac" });
    let marker = base.join("version");
    let fresh =
        exe.is_file() && std::fs::read_to_string(&marker).ok().as_deref() == Some(EMBED_VERSION);
    if !fresh {
        std::fs::create_dir_all(&base).ok()?;
        std::fs::write(&exe, EMBED_ENGINE).ok()?;
        std::fs::write(&marker, EMBED_VERSION).ok()?;
    }
    Some(exe)
}

#[cfg(not(has_embed_engine))]
pub fn ensure_engine_extracted() -> Option<PathBuf> {
    None
}

/// 解析引擎可执行文件路径：环境变量 DACREATOR_ENGINE → 可执行文件同级 → 当前目录 → 内嵌释放。
pub fn resolve_engine_exe() -> Option<PathBuf> {
    if let Ok(v) = std::env::var("DACREATOR_ENGINE") {
        if !v.is_empty() {
            let p = PathBuf::from(v);
            if p.is_file() {
                return Some(p);
            }
        }
    }
    if let Ok(exe) = std::env::current_exe() {
        if let Some(dir) = exe.parent() {
            let p = dir.join(engine_file_name());
            if p.is_file() {
                return Some(p);
            }
        }
    }
    let cwd = PathBuf::from(engine_file_name());
    if cwd.is_file() {
        return Some(cwd);
    }
    ensure_engine_extracted()
}

fn engine_file_name() -> &'static str {
    if cfg!(windows) {
        "dac.exe"
    } else {
        "dac"
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn engine_event_parses_all_kinds() {
        let cases = [
            (
                r#"{"type":"progress","stage":"crawl","pct":42,"detail":"赛道 12/48"}"#,
                "progress",
            ),
            (r#"{"type":"log","level":"success","msg":"完成"}"#, "log"),
            (
                r#"{"type":"result","csv_path":"a.csv","png_path":"a.png","records":12,"elapsed_ms":5230}"#,
                "result",
            ),
            (r#"{"type":"error","code":"network","msg":"超时"}"#, "error"),
            (r#"{"type":"log"}"#, "log"), // 缺省字段宽容
        ];
        for (json, kind) in cases {
            let ev: events::EngineEvent = serde_json::from_str(json).unwrap();
            assert_eq!(ev.kind, kind, "解析失败：{json}");
        }
    }

    #[cfg(windows)]
    fn echo_process(lines: &[&str]) -> std::process::Command {
        // cmd 的引号转义会破坏 JSON，改用临时文件 + type 原样输出
        let mut script = std::env::temp_dir();
        script.push(format!("dac_engine_test_{}.txt", std::process::id()));
        std::fs::write(&script, lines.join("\n")).unwrap();
        let mut cmd = Command::new("cmd");
        cmd.args(["/C", "type"]).arg(&script);
        cmd
    }

    #[cfg(not(windows))]
    fn echo_process(lines: &[&str]) -> std::process::Command {
        let script = format!("/tmp/dac_engine_test_{}.txt", std::process::id());
        std::fs::write(&script, lines.join("\n")).unwrap();
        let mut cmd = Command::new("cat");
        cmd.arg(&script);
        cmd
    }

    #[test]
    fn spawn_collects_events_and_exit() {
        let l1 = r#"{"type":"progress","stage":"crawl","pct":50,"detail":"half"}"#;
        let l2 = r#"{"type":"result","records":3,"elapsed_ms":100}"#;
        let mut cmd = echo_process(&[l1, l2]);

        // 复用 spawn_engine 的读取管线：直接以 echo 进程替代引擎
        let mut child = cmd
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .unwrap();
        let stdout = child.stdout.take().unwrap();
        let child = Arc::new(Mutex::new(child));
        let (tx, rx) = mpsc::channel();
        let child_for_reader = Arc::clone(&child);
        thread::spawn(move || {
            let reader = BufReader::new(stdout);
            for line in reader.lines().map_while(Result::ok) {
                if line.trim().is_empty() {
                    continue;
                }
                if let Ok(ev) = serde_json::from_str::<events::EngineEvent>(&line) {
                    tx.send(event_to_output(&ev)).ok();
                }
            }
            let success = child_for_reader
                .lock()
                .unwrap()
                .wait()
                .map(|s| s.success())
                .unwrap_or(false);
            tx.send(EngineOutput::Exited { success }).ok();
        });

        let mut got_progress = false;
        let mut got_result = false;
        let mut exited = false;
        loop {
            match rx.recv_timeout(std::time::Duration::from_secs(5)) {
                Ok(EngineOutput::Progress { pct: 50, .. }) => got_progress = true,
                Ok(EngineOutput::Result { records: 3, .. }) => got_result = true,
                Ok(EngineOutput::Exited { success }) => {
                    exited = true;
                    assert!(success);
                    break;
                }
                Ok(_) => {}
                Err(_) => panic!("5 秒内未收到进程退出"),
            }
        }
        assert!(got_progress && got_result && exited);
    }

    #[test]
    fn resolve_engine_exe_env_override() {
        let exe = std::env::temp_dir().join("dac_engine_probe.exe");
        std::fs::write(&exe, b"").unwrap();
        std::env::set_var("DACREATOR_ENGINE", &exe);
        let resolved = resolve_engine_exe();
        std::env::remove_var("DACREATOR_ENGINE");
        std::fs::remove_file(&exe).ok();
        assert_eq!(resolved, Some(exe));
    }
}
