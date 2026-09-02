// DACreator GUI — Metro Design 主题系统
// 移植自 PezMax-One：扁平、大字体、内容优先、色块分区，
// 深浅色与强调色切换均由 MetroAnim 驱动平滑过渡。

use sokuou::{EasingMode, MetroAnim, UwpEasing};
use std::cell::{Cell, RefCell};
use std::path::PathBuf;

use egui::{FontFamily, FontId, TextStyle, Vec2};

// ── 外观模式 ─────────────────────────────────────────────────────────────────

#[derive(Clone, Copy, PartialEq, Debug, Default)]
pub enum ThemeMode {
    Light,
    Dark,
    #[default]
    System,
}

// ── 强调色预设 ───────────────────────────────────────────────────────────────

pub struct AccentPreset {
    pub name: &'static str,
    pub r: u8,
    pub g: u8,
    pub b: u8,
}

pub const ACCENT_PRESETS: &[AccentPreset] = &[
    AccentPreset {
        name: "钴蓝",
        r: 0x3B,
        g: 0x82,
        b: 0xF6,
    },
    AccentPreset {
        name: "云杉",
        r: 0x1D,
        g: 0xB9,
        b: 0x54,
    },
    AccentPreset {
        name: "绯红",
        r: 0xEF,
        g: 0x44,
        b: 0x44,
    },
    AccentPreset {
        name: "琥珀",
        r: 0xF5,
        g: 0x9E,
        b: 0x0B,
    },
    AccentPreset {
        name: "堇紫",
        r: 0x8B,
        g: 0x5C,
        b: 0xF6,
    },
];

// ── 线程局部全局状态 ─────────────────────────────────────────────────────────

thread_local! {
    static IS_DARK:     Cell<bool>  = const { Cell::new(false) };
    static ACCENT_IDX:  Cell<usize> = const { Cell::new(0) };
    static ACCENT_TRANSITION: RefCell<AccentTransition> = RefCell::new(AccentTransition::idle());
    static DARK_TRANSITION:   RefCell<DarkTransition>   = RefCell::new(DarkTransition::idle());
}

pub fn set_dark(dark: bool) {
    IS_DARK.with(|d| d.set(dark));
}

pub fn is_dark() -> bool {
    IS_DARK.with(|d| d.get())
}

pub fn set_accent(idx: usize) {
    ACCENT_IDX.with(|i| i.set(idx.min(ACCENT_PRESETS.len().saturating_sub(1))));
}

pub fn accent_idx() -> usize {
    ACCENT_IDX.with(|i| i.get())
}

// ── 强调色过渡动画（MetroAnim 驱动）────────────────────────────────────────

struct AccentTransition {
    anim: MetroAnim,
    from_r: u8,
    from_g: u8,
    from_b: u8,
    to_r: u8,
    to_g: u8,
    to_b: u8,
    active: bool,
}

impl AccentTransition {
    fn idle() -> Self {
        Self {
            anim: MetroAnim::new(0.3, UwpEasing::Quadratic, EasingMode::EaseOut),
            from_r: 0,
            from_g: 0,
            from_b: 0,
            to_r: 0,
            to_g: 0,
            to_b: 0,
            active: false,
        }
    }
}

/// 开始强调色过渡（可中断，从中断处继续）。
pub fn start_accent_transition(new_idx: usize) {
    let new_idx = new_idx.min(ACCENT_PRESETS.len().saturating_sub(1));
    let new_p = &ACCENT_PRESETS[new_idx];
    ACCENT_TRANSITION.with(|t| {
        let mut t = t.borrow_mut();
        let (cr, cg, cb) = if t.active {
            interp_rgb(
                t.anim.value(),
                (t.from_r, t.from_g, t.from_b),
                (t.to_r, t.to_g, t.to_b),
            )
        } else {
            let old_p = &ACCENT_PRESETS[accent_idx()];
            (old_p.r, old_p.g, old_p.b)
        };
        t.from_r = cr;
        t.from_g = cg;
        t.from_b = cb;
        t.to_r = new_p.r;
        t.to_g = new_p.g;
        t.to_b = new_p.b;
        t.anim.jump_to(0.0);
        t.anim.set_target(1.0);
        t.active = true;
    });
}

pub fn update_accent_transition(dt: f64) {
    ACCENT_TRANSITION.with(|t| {
        let mut t = t.borrow_mut();
        if t.active {
            t.anim.update(dt);
            if t.anim.is_steady() {
                t.active = false;
            }
        }
    });
}

pub fn is_transitioning() -> bool {
    ACCENT_TRANSITION.with(|t| t.borrow().active)
}

fn current_accent_rgb() -> (u8, u8, u8) {
    ACCENT_TRANSITION.with(|t| {
        let t = t.borrow();
        if t.active {
            interp_rgb(
                t.anim.value(),
                (t.from_r, t.from_g, t.from_b),
                (t.to_r, t.to_g, t.to_b),
            )
        } else {
            let p = &ACCENT_PRESETS[accent_idx()];
            (p.r, p.g, p.b)
        }
    })
}

fn interp_rgb(v: f64, from: (u8, u8, u8), to: (u8, u8, u8)) -> (u8, u8, u8) {
    let f = |a: u8, b: u8| {
        (a as f64 + (b as f64 - a as f64) * v)
            .round()
            .clamp(0.0, 255.0) as u8
    };
    (f(from.0, to.0), f(from.1, to.1), f(from.2, to.2))
}

// ── 深浅色过渡动画（MetroAnim 驱动）────────────────────────────────────────

struct DarkTransition {
    anim: MetroAnim,
    active: bool,
    target_dark: bool,
}

impl DarkTransition {
    fn idle() -> Self {
        Self {
            anim: MetroAnim::new(0.3, UwpEasing::Quadratic, EasingMode::EaseOut),
            active: false,
            target_dark: false,
        }
    }
}

pub fn start_dark_transition(target_dark: bool) {
    DARK_TRANSITION.with(|t| {
        let mut t = t.borrow_mut();
        t.anim.jump_to(0.0);
        t.anim.set_target(1.0);
        t.target_dark = target_dark;
        t.active = true;
    });
}

pub fn update_dark_transition(dt: f64) {
    DARK_TRANSITION.with(|t| {
        let mut t = t.borrow_mut();
        if t.active {
            t.anim.update(dt);
            if t.anim.is_steady() {
                t.active = false;
            }
        }
    });
}

pub fn is_dark_transitioning() -> bool {
    DARK_TRANSITION.with(|t| t.borrow().active)
}

fn dark_progress() -> f64 {
    DARK_TRANSITION.with(|t| {
        let t = t.borrow();
        if t.active {
            if t.target_dark {
                t.anim.value()
            } else {
                1.0 - t.anim.value()
            }
        } else if is_dark() {
            1.0
        } else {
            0.0
        }
    })
}

/// ThemeMode::System 时跟随系统主题；显式模式直接生效。
pub fn effective_dark(ctx: &egui::Context, mode: ThemeMode) -> bool {
    match mode {
        ThemeMode::Light => false,
        ThemeMode::Dark => true,
        ThemeMode::System => ctx.system_theme() == Some(egui::Theme::Dark),
    }
}

// ── 调色板（函数化，每帧读取插值状态）──────────────────────────────────────

pub mod colors {
    use super::current_accent_rgb;
    use super::dark_progress;
    use egui::Color32;

    fn lerp_dark(light: Color32, dark: Color32) -> Color32 {
        let t = dark_progress();
        if t <= 0.0 {
            return light;
        }
        if t >= 1.0 {
            return dark;
        }
        let f = |a: u8, b: u8| {
            (a as f64 + (b as f64 - a as f64) * t)
                .round()
                .clamp(0.0, 255.0) as u8
        };
        Color32::from_rgb(
            f(light.r(), dark.r()),
            f(light.g(), dark.g()),
            f(light.b(), dark.b()),
        )
    }

    pub fn primary() -> Color32 {
        let (r, g, b) = current_accent_rgb();
        Color32::from_rgb(r, g, b)
    }

    pub fn primary_light() -> Color32 {
        let (r, g, b) = current_accent_rgb();
        Color32::from_rgb(
            (r as u16 * 7 / 10 + 255 * 3 / 10) as u8,
            (g as u16 * 7 / 10 + 255 * 3 / 10) as u8,
            (b as u16 * 7 / 10 + 255 * 3 / 10) as u8,
        )
    }

    // ── 文本 ────────────────────────────────────────────────────────────────
    pub fn text_primary() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0x1A, 0x1A, 0x2E),
            Color32::from_rgb(0xF0, 0xF0, 0xF2),
        )
    }
    pub fn text_secondary() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0x60, 0x60, 0x70),
            Color32::from_rgb(0x9A, 0x9A, 0xAA),
        )
    }

    // ── 背景 ────────────────────────────────────────────────────────────────
    pub fn bg_white() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0xF5, 0xF0, 0xE8),
            Color32::from_rgb(0x1C, 0x1C, 0x1C),
        )
    }
    pub fn bg_card() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0xFA, 0xF7, 0xF0),
            Color32::from_rgb(0x26, 0x26, 0x26),
        )
    }
    pub fn bg_sidebar() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0xC0, 0xB0, 0x9C),
            Color32::from_rgb(0x14, 0x14, 0x1E),
        )
    }
    pub fn bg_hover() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0xE8, 0xE0, 0xD5),
            Color32::from_rgb(0x2E, 0x2E, 0x3A),
        )
    }
    pub fn bg_selected() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0xDD, 0xD3, 0xC5),
            Color32::from_rgb(0x35, 0x35, 0x5A),
        )
    }
    pub fn bg_input() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0xEE, 0xE8, 0xDE),
            Color32::from_rgb(0x20, 0x20, 0x20),
        )
    }

    pub fn border() -> Color32 {
        lerp_dark(
            Color32::from_rgb(0xD4, 0xC8, 0xB8),
            Color32::from_rgb(0x3A, 0x3A, 0x3A),
        )
    }

    // ── 状态色（固定）──────────────────────────────────────────────────────
    pub fn success() -> Color32 {
        Color32::from_rgb(0x00, 0xBC, 0x70)
    }
    pub fn warning() -> Color32 {
        Color32::from_rgb(0xF7, 0x63, 0x00)
    }
    pub fn error() -> Color32 {
        Color32::from_rgb(0xE8, 0x11, 0x23)
    }
}

// ── 字体与样式 ───────────────────────────────────────────────────────────────

/// 加载 CJK 字体：优先使用随包资产（Noto Sans CJK，渲染结果确定），
/// 其次系统字体。`asset_font` 传 None 时仅尝试系统字体。
pub fn setup_fonts(ctx: &egui::Context, asset_font: Option<PathBuf>) {
    let mut fonts = egui::FontDefinitions::default();

    let mut load = |bytes: Vec<u8>, index: u32, name: &str| {
        fonts.font_data.insert(
            name.to_owned(),
            std::sync::Arc::new({
                let mut fd = egui::FontData::from_owned(bytes);
                fd.index = index;
                fd
            }),
        );
        let prop = fonts.families.entry(FontFamily::Proportional).or_default();
        if !prop.iter().any(|f| f == name) {
            prop.insert(0, name.to_owned());
        }
        let mono = fonts.families.entry(FontFamily::Monospace).or_default();
        if !mono.iter().any(|f| f == name) {
            mono.insert(0, name.to_owned());
        }
    };

    let mut loaded = false;
    if let Some(path) = asset_font {
        if let Ok(bytes) = std::fs::read(&path) {
            load(bytes, 0, "bundled_noto");
            loaded = true;
        }
    }
    if !loaded {
        let system_candidates: &[&str] = &[
            "C:\\Windows\\Fonts\\msyh.ttc",
            "C:\\Windows\\Fonts\\simhei.ttf",
            "/System/Library/Fonts/PingFang.ttc",
            "/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
            "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
        ];
        for path in system_candidates {
            if let Ok(bytes) = std::fs::read(path) {
                load(bytes, 0, "system_cjk");
                loaded = true;
                break;
            }
        }
    }
    if !loaded {
        eprintln!("⚠️ 未找到 CJK 字体，中文可能显示为方块");
    }

    ctx.set_fonts(fonts);
}

fn metro_text_styles() -> BTreeMap<TextStyle, FontId> {
    use TextStyle::*;
    [
        (Heading, FontId::new(28.0, FontFamily::Proportional)),
        (
            Name("h1".into()),
            FontId::new(24.0, FontFamily::Proportional),
        ),
        (
            Name("h2".into()),
            FontId::new(20.0, FontFamily::Proportional),
        ),
        (
            Name("h3".into()),
            FontId::new(16.0, FontFamily::Proportional),
        ),
        (Body, FontId::new(14.0, FontFamily::Proportional)),
        (Monospace, FontId::new(13.0, FontFamily::Monospace)),
        (Button, FontId::new(14.0, FontFamily::Proportional)),
        (Small, FontId::new(12.0, FontFamily::Proportional)),
    ]
    .into()
}

use std::collections::BTreeMap;

/// 应用 Metro 主题到 egui 上下文（过渡期间每帧调用）。
pub fn apply_metro_theme(ctx: &egui::Context) {
    let dark = is_dark();
    let border_color = colors::border();
    let text_fg = colors::text_primary();
    let text_weak = colors::text_secondary();
    let bg_white = colors::bg_white();
    let bg_card = colors::bg_card();
    let bg_input = colors::bg_input();
    let bg_hover = colors::bg_hover();
    let bg_selected = colors::bg_selected();
    let primary = colors::primary();
    let primary_light = colors::primary_light();
    let warning = colors::warning();
    let error = colors::error();
    let zero = egui::CornerRadius::same(0);

    ctx.style_mut(|style| {
        style.text_styles = metro_text_styles();
        style.spacing.item_spacing = Vec2::new(12.0, 8.0);
        style.spacing.button_padding = Vec2::new(16.0, 8.0);
        style.spacing.indent = 24.0;

        style.visuals = if dark {
            egui::Visuals::dark()
        } else {
            egui::Visuals::light()
        };
        style.visuals.dark_mode = dark;

        style.visuals.override_text_color = None;
        style.visuals.hyperlink_color = primary;
        style.visuals.selection.stroke = egui::Stroke {
            width: 1.0,
            color: primary,
        };
        style.visuals.selection.bg_fill = bg_selected;
        style.visuals.warn_fg_color = warning;
        style.visuals.error_fg_color = error;

        style.visuals.window_fill = bg_white;
        style.visuals.panel_fill = bg_white;
        style.visuals.faint_bg_color = bg_card;
        style.visuals.extreme_bg_color = bg_card;
        style.visuals.code_bg_color = bg_input;

        style.visuals.window_corner_radius = zero;
        style.visuals.menu_corner_radius = zero;

        style.visuals.widgets.noninteractive = egui::style::WidgetVisuals {
            bg_fill: bg_card,
            weak_bg_fill: bg_input,
            bg_stroke: egui::Stroke::NONE,
            fg_stroke: egui::Stroke::new(1.0, text_weak),
            corner_radius: zero,
            expansion: 0.0,
        };
        style.visuals.widgets.inactive = egui::style::WidgetVisuals {
            bg_fill: bg_card,
            weak_bg_fill: bg_white,
            bg_stroke: egui::Stroke::new(1.0, border_color),
            fg_stroke: egui::Stroke::new(1.5, text_fg),
            corner_radius: zero,
            expansion: 0.0,
        };
        style.visuals.widgets.hovered = egui::style::WidgetVisuals {
            bg_fill: bg_hover,
            weak_bg_fill: bg_hover,
            bg_stroke: egui::Stroke::new(1.0, primary_light),
            fg_stroke: egui::Stroke::new(1.5, text_fg),
            corner_radius: zero,
            expansion: 0.0,
        };
        style.visuals.widgets.active = egui::style::WidgetVisuals {
            bg_fill: bg_selected,
            weak_bg_fill: bg_selected,
            bg_stroke: egui::Stroke::new(1.0, primary),
            fg_stroke: egui::Stroke::new(2.0, text_fg),
            corner_radius: zero,
            expansion: 0.0,
        };
        style.visuals.widgets.open = egui::style::WidgetVisuals {
            bg_fill: bg_selected,
            weak_bg_fill: bg_selected,
            bg_stroke: egui::Stroke::new(1.0, primary),
            fg_stroke: egui::Stroke::new(1.5, text_fg),
            corner_radius: zero,
            expansion: 0.0,
        };
    });
}
