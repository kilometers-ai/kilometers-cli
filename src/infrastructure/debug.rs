use std::sync::OnceLock;

static VERBOSE_MODE: OnceLock<bool> = OnceLock::new();

pub fn set_verbose(enabled: bool) {
    VERBOSE_MODE.set(enabled).ok();
}

pub fn is_verbose() -> bool {
    *VERBOSE_MODE.get().unwrap_or(&false)
}

#[macro_export]
macro_rules! debug_log {
    ($($arg:tt)*) => {
        if $crate::infrastructure::debug::is_verbose() {
            eprintln!("[DEBUG] {}", format!($($arg)*));
        }
    };
}

pub use debug_log;