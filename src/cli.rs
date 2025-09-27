use clap::{Parser, Subcommand};
use std::path::PathBuf;

#[derive(Parser, Debug)]
#[command(name = "km")]
#[command(author, version, about = "Official Kilometers CLI proxy for MCP servers", long_about = None)]
pub struct Cli {
    /// Verbose mode (-v, -vv, -vvv)
    #[arg(short, long, action = clap::ArgAction::Count)]
    pub verbose: u8,

    /// Config file path
    #[arg(short, long, default_value = "km_config.json")]
    pub config: PathBuf,

    #[command(subcommand)]
    pub command: Commands,
}

#[derive(Subcommand, Debug)]
pub enum Commands {
    /// Initialize configuration with API key
    Init {
        /// API key for Kilometers.ai (or set KM_API_KEY env var)
        #[arg(short, long)]
        api_key: Option<String>,

        /// API base URL
        #[arg(long, default_value = "https://api.kilometers.ai")]
        api_url: String,
    },

    /// Monitor and proxy MCP requests
    Monitor {
        /// Command and arguments to proxy (everything after --)
        #[arg(trailing_var_arg = true, allow_hyphen_values = true, required = true)]
        args: Vec<String>,

        /// Skip risk analysis filters (local logging only)
        #[arg(long)]
        local_only: bool,

        /// Override user tier for testing
        #[arg(long, hide = true)]
        override_tier: Option<String>,

        /// Log file for MCP traffic
        #[arg(long, default_value = "mcp_traffic.jsonl")]
        log_file: PathBuf,
    },

    /// Clear all logs
    ClearLogs {
        /// Also clear config file
        #[arg(long)]
        include_config: bool,
    },

    /// Show current configuration
    Config {
        /// Show API key (hidden by default)
        #[arg(long)]
        show_secrets: bool,
    },

    /// Analyze log files
    Logs {
        /// Log file to analyze
        #[arg(short, long, default_value = "mcp_traffic.jsonl")]
        file: PathBuf,

        /// Show only requests
        #[arg(long, conflicts_with = "responses")]
        requests: bool,

        /// Show only responses
        #[arg(long)]
        responses: bool,

        /// Filter by method name
        #[arg(short, long)]
        method: Option<String>,

        /// Tail mode - follow log file
        #[arg(short, long)]
        tail: bool,

        /// Number of lines to show (default: all)
        #[arg(short = 'n', long)]
        lines: Option<usize>,
    },
}

impl Cli {
    pub fn get_log_level(&self) -> tracing::Level {
        match self.verbose {
            0 => tracing::Level::ERROR,
            1 => tracing::Level::WARN,
            2 => tracing::Level::INFO,
            3 => tracing::Level::DEBUG,
            _ => tracing::Level::TRACE,
        }
    }
}
