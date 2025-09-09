// anyhow provides flexible error handling
use crate::domain::proxy::ProxyCommand;
use crate::infrastructure::log_repository::LogRepository;
use anyhow::Result;

// Unit struct (no fields) representing the Init command
pub struct InitCommand;

impl InitCommand {
    // Static async method (no self parameter)
    // Executes the initialization command
    pub async fn execute() -> Result<()> {
        // Create a new instance of the authentication service
        let service = crate::application::services::AuthenticationService::new();
        // Call async method and await its completion
        // The ? operator propagates any errors up
        service.initialize_configuration().await
    }
}

// Struct with a private field (no `pub`)
pub struct MonitorCommand {
    command: ProxyCommand,
}

impl MonitorCommand {
    // Constructor that creates a ProxyCommand internally
    pub fn new(command: String, args: Vec<String>) -> Self {
        Self {
            // Create ProxyCommand from the provided arguments
            command: ProxyCommand::new(command, args),
        }
    }

    // Takes ownership of self (consumes the struct)
    // This means the MonitorCommand can't be used after calling execute
    pub async fn execute(self) -> Result<()> {
        let service = crate::application::services::ProxyService::new();
        // Move self.command into the service method
        service.run_proxy(self.command).await
    }
}

// Unit struct for clearing logs command
pub struct ClearLogsCommand;

impl ClearLogsCommand {
    // Static async method to execute log clearing
    pub async fn execute() -> Result<()> {
        let log_repo = LogRepository::new();
        log_repo.clear_logs().await
    }
}
