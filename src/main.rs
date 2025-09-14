// Module declarations - tells Rust about our module structure
// Like namespace declarations or #include in C/C++
mod application; // Use cases and workflows
mod domain; // Business logic and entities
mod infrastructure; // External dependencies and I/O

use anyhow::Result; // Flexible error type (like std::exception hierarchy)
use clap::{Arg, Command}; // Command-line argument parsing library
                          // Import specific types from our application layer
use application::commands::{ClearLogsCommand, InitCommand, MonitorCommand};

// Attribute macro that sets up the async runtime
// Like [STAThread] in C# - modifies how the function runs
#[tokio::main]
// Main function returns Result - can propagate errors with ?
// Like int main() in C++ but with error handling built-in
async fn main() -> Result<()> {
    // Builder pattern for constructing CLI interface
    // Similar to CommandLineParser or ArgumentParser in other languages
    let matches = Command::new("km") // Program name
        .about("MCP Proxy CLI") // Description
        .subcommand(
            // Add a subcommand
            Command::new("monitor")
                .about("Monitor and proxy MCP requests")
                .arg(
                    Arg::new("verbose")
                        .short('v')
                        .long("verbose")
                        .help("Enable verbose output")
                        .action(clap::ArgAction::SetTrue),
                )
                .arg(
                    // Define an argument
                    Arg::new("command")
                        .help("The command to proxy to")
                        .action(clap::ArgAction::Append) // Can appear multiple times
                        .required(true) // Must be provided
                        .last(true), // Takes remaining args
                ),
        )
        .subcommand(Command::new("init").about("Initialize configuration with API key"))
        .subcommand(Command::new("clear-logs").about("Clear all logged requests and responses"))
        .get_matches(); // Parse command line arguments

    // Pattern matching on subcommand (like switch in C++ but more powerful)
    // Returns Option<(&str, &ArgMatches)> - either Some(subcommand) or None
    match matches.subcommand() {
        // Destructure tuple: extract subcommand name and its arguments
        Some(("monitor", monitor_matches)) => {
            // Check if verbose flag is set
            let verbose = monitor_matches.get_flag("verbose");

            // Extract command arguments as Vec<String>
            // Iterator chain: get -> unwrap -> map -> collect (like LINQ in C#)
            let command_args: Vec<String> = monitor_matches
                .get_many::<String>("command") // Get iterator over values
                .unwrap() // Panic if None (we know it exists)
                .map(|s| s.to_string()) // Convert &String to String
                .collect(); // Collect into Vec

            // Validation logic
            if command_args.is_empty() {
                eprintln!("Usage: km monitor [--verbose] -- <command> [args...]"); // Print to stderr
                std::process::exit(1); // Exit with error code (like return 1 in C++)
            }

            // Handle optional '--' separator
            let start_index = if command_args[0] == "--" { 1 } else { 0 };

            if start_index >= command_args.len() {
                eprintln!("Usage: km monitor [--verbose] -- <command> [args...]");
                std::process::exit(1);
            }

            let actual_command = command_args[start_index].clone(); // Clone to get owned String
                                                                    // Array slice syntax: [start..end] (like substring in other languages)
            let actual_args = command_args[start_index + 1..].to_vec();

            if verbose {
                eprintln!("Starting MCP proxy with command: {} {:?}", actual_command, actual_args);
            }

            // Create and execute command
            let command = MonitorCommand::new(actual_command, actual_args);
            command.execute().await?; // Await async execution, ? propagates errors
        }
        Some(("init", _)) => {
            // _ ignores the ArgMatches (we don't need them)
            InitCommand::execute().await?;
        }
        Some(("clear-logs", _)) => {
            ClearLogsCommand::execute().await?;
        }
        _ => {
            // Default case - no subcommand matched
            eprintln!("Use 'km monitor [--verbose] -- <command> [args...]' to start proxying, 'km init' to setup configuration, or 'km clear-logs' to clear logs");
            std::process::exit(1);
        }
    }

    Ok(()) // Return success (unit type wrapped in Ok)
} // End of main function
