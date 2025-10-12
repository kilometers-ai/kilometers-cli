use anyhow::Result;
use clap::Parser;

mod auth;
mod cli;
mod config;
mod device_auth;
mod filters;
mod handlers;
mod keyring_token_store;
mod proxy;

use cli::{Cli, Commands, DoctorCommands};

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    // Initialize logging with verbosity level
    tracing_subscriber::fmt()
        .with_max_level(cli.get_log_level())
        .init();

    tracing::debug!("Starting km cli with command: {:?}", cli.command);

    match cli.command {
        Commands::Init { api_key, api_url } => {
            handlers::handle_init(&cli.config, api_key, api_url).await?
        }
        Commands::Monitor {
            args,
            local_only,
            override_tier,
            log_file,
        } => {
            handlers::handle_monitor(&cli.config, args, local_only, override_tier, log_file).await?
        }
        Commands::ClearLogs { include_config } => {
            handlers::handle_clear_logs(include_config, &cli.config)?
        }
        Commands::Config { show_secrets } => {
            handlers::handle_show_config(&cli.config, show_secrets)?
        }
        Commands::Logs {
            file,
            requests,
            responses,
            method,
            tail,
            lines,
        } => handlers::handle_logs(file, requests, responses, method, tail, lines)?,
        Commands::Doctor { command } => handle_doctor(command)?,
    }

    Ok(())
}

fn handle_doctor(command: DoctorCommands) -> Result<()> {
    match command {
        DoctorCommands::Jwt => handlers::handle_doctor_jwt(),
    }
}
