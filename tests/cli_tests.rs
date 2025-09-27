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
