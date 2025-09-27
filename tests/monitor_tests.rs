use clap::Parser;
use km::cli::{Cli, Commands};

#[test]
fn test_monitor_command_parsing() {
    let args = vec![
        "km",
        "monitor",
        "--",
        "npx",
        "-y",
        "@modelcontextprotocol/server-filesystem",
        "~/Documents",
    ];

    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Monitor {
            args, local_only, ..
        } => {
            assert_eq!(args.len(), 4);
            assert_eq!(args[0], "npx");
            assert_eq!(args[1], "-y");
            assert_eq!(args[2], "@modelcontextprotocol/server-filesystem");
            assert_eq!(args[3], "~/Documents");
            assert!(!local_only);
        }
        _ => panic!("Expected Monitor command"),
    }
}

#[test]
fn test_monitor_command_with_local_only_flag() {
    let args = vec!["km", "monitor", "--local-only", "--", "some-mcp-server"];

    let cli = Cli::parse_from(args);

    match cli.command {
        Commands::Monitor {
            args, local_only, ..
        } => {
            assert_eq!(args.len(), 1);
            assert_eq!(args[0], "some-mcp-server");
            assert!(local_only);
        }
        _ => panic!("Expected Monitor command"),
    }
}
