use crate::domain::proxy::LogEntry;
use anyhow::Result; // Error handling
use std::fs::{create_dir_all, remove_file, OpenOptions}; // File operations
use std::io::Write; // Trait for writing to streams (like IOutputStream in C#)
use std::path::PathBuf;

// Simple repository for writing log entries to file
#[derive(Debug)]
pub struct LogRepository {
    log_path: String, // File path as owned String
}

impl Default for LogRepository {
    fn default() -> Self {
        Self::new()
    }
}

impl LogRepository {
    // Constructor using standard data/log directory
    pub fn new() -> Self {
        let log_dir = dirs::data_local_dir()
            .or_else(dirs::data_dir)
            .map(|dir| dir.join("km"))
            .unwrap_or_else(|| PathBuf::from(".").join(".local").join("share").join("km"));

        let log_path = log_dir.join("mcp_proxy.log");

        Self {
            log_path: log_path.to_string_lossy().to_string(),
        }
    }

    // Takes LogEntry by reference (borrows, doesn't consume it)
    // Like passing const T& in C++ or ref T in C#
    pub fn write_entry(&self, entry: &LogEntry) -> Result<()> {
        // Create log directory if it doesn't exist
        if let Some(parent) = PathBuf::from(&self.log_path).parent() {
            create_dir_all(parent)?;
        }

        // Builder pattern with method chaining
        // Similar to FileStream constructor options in C#
        let mut log_file = OpenOptions::new()
            .create(true) // Create file if it doesn't exist
            .append(true) // Append to end of file (don't overwrite)
            .open(&self.log_path)?; // Open file with these options

        // writeln! is like fprintf or Console.WriteLine
        writeln!(log_file, "{}", entry.format_for_log())?;
        Ok(()) // Return success
    }

    // Clear all logs by removing the log file
    pub async fn clear_logs(&self) -> Result<()> {
        match remove_file(&self.log_path) {
            Ok(()) => {
                println!("Successfully cleared logs from: {}", self.log_path);
                Ok(())
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                println!("No log file found at: {} (already clear)", self.log_path);
                Ok(())
            }
            Err(e) => Err(anyhow::anyhow!("Failed to clear logs: {}", e)),
        }
    }
}
