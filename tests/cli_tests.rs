use clap::Parser;
use km::cli::{Cli, Commands};
use std::path::PathBuf;

#[test]
fn test_init_command_with_api_key() {
    let args = vec!["km", "init", "--api-key", "test-key-123"];

    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Init { api_key, api_url } => {
            assert_eq!(api_key, Some("test-key-123".to_string()));
            assert_eq!(api_url, "https://api.kilometers.ai");
        }
        _ => panic!("Expected Init command"),
    }
}

#[test]
fn test_init_command_with_custom_api_url() {
    let args = vec![
        "km",
        "init",
        "--api-key",
        "test-key",
        "--api-url",
        "https://custom.api.com",
    ];

    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Init { api_key, api_url } => {
            assert_eq!(api_key, Some("test-key".to_string()));
            assert_eq!(api_url, "https://custom.api.com");
        }
        _ => panic!("Expected Init command"),
    }
}

#[test]
fn test_init_command_without_api_key() {
    let args = vec!["km", "init"];

    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Init { api_key, api_url } => {
            assert_eq!(api_key, None);
            assert_eq!(api_url, "https://api.kilometers.ai");
        }
        _ => panic!("Expected Init command"),
    }
}

#[test]
fn test_clear_logs_command() {
    let args = vec!["km", "clear-logs"];

    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::ClearLogs { include_config } => {
            assert!(!include_config);
        }
        _ => panic!("Expected ClearLogs command"),
    }
}

#[test]
fn test_clear_logs_with_include_config() {
    let args = vec!["km", "clear-logs", "--include-config"];

    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::ClearLogs { include_config } => {
            assert!(include_config);
        }
        _ => panic!("Expected ClearLogs command"),
    }
}

#[test]
fn test_verbose_flag_single() {
    let args = vec!["km", "-v", "init"];

    let cli = Cli::parse_from(args);
    assert_eq!(cli.verbose, 1);
}

#[test]
fn test_verbose_flag_multiple() {
    let args = vec!["km", "-vvv", "init"];

    let cli = Cli::parse_from(args);
    assert_eq!(cli.verbose, 3);
}

#[test]
fn test_custom_config_path() {
    let args = vec!["km", "--config", "/custom/path/config.json", "init"];

    let cli = Cli::parse_from(args);
    assert_eq!(cli.config, PathBuf::from("/custom/path/config.json"));
}

#[test]
fn test_default_config_path() {
    let args = vec!["km", "init"];

    let cli = Cli::parse_from(args);
    assert_eq!(cli.config, PathBuf::from("km_config.json"));
}

#[test]
fn test_get_log_level_error() {
    let args = vec!["km", "init"];
    let cli = Cli::parse_from(args);
    assert_eq!(cli.get_log_level(), tracing::Level::ERROR);
}

#[test]
fn test_get_log_level_warn() {
    let args = vec!["km", "-v", "init"];
    let cli = Cli::parse_from(args);
    assert_eq!(cli.get_log_level(), tracing::Level::WARN);
}

#[test]
fn test_get_log_level_info() {
    let args = vec!["km", "-vv", "init"];
    let cli = Cli::parse_from(args);
    assert_eq!(cli.get_log_level(), tracing::Level::INFO);
}

#[test]
fn test_get_log_level_debug() {
    let args = vec!["km", "-vvv", "init"];
    let cli = Cli::parse_from(args);
    assert_eq!(cli.get_log_level(), tracing::Level::DEBUG);
}

#[test]
fn test_get_log_level_trace() {
    let args = vec!["km", "-vvvv", "init"];
    let cli = Cli::parse_from(args);
    assert_eq!(cli.get_log_level(), tracing::Level::TRACE);
}

#[test]
fn test_monitor_command_basic() {
    let args = vec!["km", "monitor", "--", "npx", "server"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Monitor {
            args,
            local_only,
            override_tier,
            log_file,
        } => {
            assert_eq!(args, vec!["npx", "server"]);
            assert!(!local_only);
            assert_eq!(override_tier, None);
            assert_eq!(log_file, PathBuf::from("mcp_traffic.jsonl"));
        }
        _ => panic!("Expected Monitor command"),
    }
}

#[test]
fn test_monitor_command_with_local_only() {
    let args = vec!["km", "monitor", "--local-only", "--", "echo", "test"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Monitor { local_only, .. } => {
            assert!(local_only);
        }
        _ => panic!("Expected Monitor command"),
    }
}

#[test]
fn test_monitor_command_with_custom_log_file() {
    let args = vec![
        "km",
        "monitor",
        "--log-file",
        "custom.log",
        "--",
        "echo",
        "test",
    ];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Monitor { log_file, .. } => {
            assert_eq!(log_file, PathBuf::from("custom.log"));
        }
        _ => panic!("Expected Monitor command"),
    }
}

#[test]
fn test_config_command_basic() {
    let args = vec!["km", "config"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Config { show_secrets } => {
            assert!(!show_secrets);
        }
        _ => panic!("Expected Config command"),
    }
}

#[test]
fn test_config_command_with_show_secrets() {
    let args = vec!["km", "config", "--show-secrets"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Config { show_secrets } => {
            assert!(show_secrets);
        }
        _ => panic!("Expected Config command"),
    }
}

#[test]
fn test_logs_command_basic() {
    let args = vec!["km", "logs"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Logs {
            file,
            requests,
            responses,
            method,
            tail,
            lines,
        } => {
            assert_eq!(file, PathBuf::from("mcp_traffic.jsonl"));
            assert!(!requests);
            assert!(!responses);
            assert_eq!(method, None);
            assert!(!tail);
            assert_eq!(lines, None);
        }
        _ => panic!("Expected Logs command"),
    }
}

#[test]
fn test_logs_command_with_requests_filter() {
    let args = vec!["km", "logs", "--requests"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Logs { requests, .. } => {
            assert!(requests);
        }
        _ => panic!("Expected Logs command"),
    }
}

#[test]
fn test_logs_command_with_responses_filter() {
    let args = vec!["km", "logs", "--responses"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Logs { responses, .. } => {
            assert!(responses);
        }
        _ => panic!("Expected Logs command"),
    }
}

#[test]
fn test_logs_command_with_method_filter() {
    let args = vec!["km", "logs", "--method", "initialize"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Logs { method, .. } => {
            assert_eq!(method, Some("initialize".to_string()));
        }
        _ => panic!("Expected Logs command"),
    }
}

#[test]
fn test_logs_command_with_tail() {
    let args = vec!["km", "logs", "--tail"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Logs { tail, .. } => {
            assert!(tail);
        }
        _ => panic!("Expected Logs command"),
    }
}

#[test]
fn test_logs_command_with_lines_limit() {
    let args = vec!["km", "logs", "--lines", "10"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Logs { lines, .. } => {
            assert_eq!(lines, Some(10));
        }
        _ => panic!("Expected Logs command"),
    }
}

#[test]
fn test_logs_command_custom_file() {
    let args = vec!["km", "logs", "--file", "custom_traffic.log"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Logs { file, .. } => {
            assert_eq!(file, PathBuf::from("custom_traffic.log"));
        }
        _ => panic!("Expected Logs command"),
    }
}

#[test]
fn test_doctor_jwt_command() {
    let args = vec!["km", "doctor", "jwt"];
    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Doctor { command } => match command {
            km::cli::DoctorCommands::Jwt => {
                // Command parsed correctly
            }
        },
        _ => panic!("Expected Doctor command"),
    }
}
