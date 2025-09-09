use anyhow::Result;
use colored::*; // Import all items from colored crate (like using namespace in C++)
use std::io::{self, Write};
// Glob imports (like `using namespace` in C++ or `using static` in C#)
use crate::domain::{auth::*, proxy::*};
use crate::infrastructure::{
    api_client::ApiClient, configuration_repository::ConfigurationRepository,
    log_repository::LogRepository, process_manager::ProcessManager,
};

// Struct is like a C++ class/C# class but with explicit composition
// Fields are stack-allocated by default (no new/malloc needed)
pub struct AuthenticationService {
    api_client: ApiClient,                      // Owned value (not a pointer)
    config_repository: ConfigurationRepository, // Also owned
}

impl Default for AuthenticationService {
    fn default() -> Self {
        Self::new()
    }
}

impl AuthenticationService {
    // Constructor (like C++ constructor or C# constructor)
    // Returns by value, not pointer (RAII - Resource Acquisition Is Initialization)
    pub fn new() -> Self {
        Self {
            // Move semantics - these values are moved into the struct
            api_client: ApiClient::new("http://localhost:5195".to_string()),
            config_repository: ConfigurationRepository::new(),
        }
    }

    // &self is like 'const this*' in C++ or 'this' in C# (immutable reference)
    // async is like C#'s async - function can be awaited
    pub async fn initialize_configuration(&self) -> Result<()> {
        println!(
            "{}",
            "Initializing kilometers CLI configuration...".bold().cyan()
        );
        println!();

        let api_key = self.prompt_for_api_key()?;

        if api_key.is_empty() {
            eprintln!("{}", "Error: API key cannot be empty".red());
            std::process::exit(1);
        }

        println!();
        println!("{}", "Validating API key...".dimmed());

        match self.validate_api_key(&api_key).await {
            Ok(customer) => {
                self.display_customer_info(&customer);
                self.config_repository
                    .save_configuration(Configuration::new(api_key))?;
                self.display_success_message();
            }
            Err(AuthenticationError::InvalidApiKey) => {
                eprintln!();
                eprintln!("{}", "✗ Invalid API key".red().bold());
                eprintln!(
                    "{}",
                    "The provided API key is not valid or has been revoked.".red()
                );
                std::process::exit(1);
            }
            Err(e) => {
                eprintln!();
                eprintln!("{} {}", "✗ Authentication failed:".red().bold(), e);
                std::process::exit(1);
            }
        }

        Ok(())
    }

    // Private method (like private in C++/C#)
    fn prompt_for_api_key(&self) -> Result<String> {
        print!("{}", "Please enter your API key: ".bold());
        io::stdout().flush()?; // ? is like throw in C# - propagates error up

        let mut api_key = String::new(); // mut = mutable (like non-const in C++)
        io::stdin().read_line(&mut api_key)?; // &mut = mutable reference (like T* in C++)
        Ok(api_key.trim().to_string()) // Return success variant of Result
    }

    // &str is like const char* in C++ - string slice (view into string data)
    // Result<Customer, AuthenticationError> is like returning either Customer or throwing AuthenticationError
    async fn validate_api_key(&self, api_key: &str) -> Result<Customer, AuthenticationError> {
        let credentials = ApiKeyCredentials::new(api_key.to_string());
        let result = self.api_client.authenticate(credentials).await?;

        // Standard if-else, but returns Result enum variants
        if result.success {
            Ok(result.customer) // Like returning success
        } else {
            Err(AuthenticationError::InvalidApiKey) // Like throwing exception
        }
    }

    fn display_customer_info(&self, customer: &Customer) {
        println!();
        println!("{}", "✓ API key validated successfully!".green().bold());
        println!();

        println!("{}", "User Information".bold().underline());
        println!("  {}: {}", "Email".bold(), customer.email.cyan());

        if let Some(org) = &customer.organization {
            println!("  {}: {}", "Organization".bold(), org.cyan());
        }

        println!("  {}: {}", "User ID".bold(), customer.id.dimmed());
        println!(
            "  {}: {}",
            "Subscription Plan".bold(),
            match customer.subscription_plan.as_str() {
                "Free" => customer.subscription_plan.yellow(),
                "Pro" => customer.subscription_plan.green(),
                "Enterprise" => customer.subscription_plan.magenta(),
                _ => customer.subscription_plan.normal(),
            }
        );
        println!(
            "  {}: {}",
            "Subscription Status".bold(),
            if customer.subscription_status == "Active" {
                customer.subscription_status.green()
            } else {
                customer.subscription_status.red()
            }
        );

        if customer.has_password {
            println!("  {}: {}", "Password Protected".bold(), "Yes".green());
        } else {
            println!("  {}: {}", "Password Protected".bold(), "No".yellow());
        }

        if let Some(last_login) = &customer.last_login_at {
            println!("  {}: {}", "Last Login".bold(), last_login.dimmed());
        }

        println!(
            "  {}: {}",
            "Account Created".bold(),
            customer.created_at.dimmed()
        );
    }

    fn display_success_message(&self) {
        let config_file = self.config_repository.get_config_path();

        println!();
        println!(
            "{} {}",
            "✓ Configuration saved to:".green(),
            config_file.bold()
        );
        println!();
        println!(
            "{}",
            "You can now use 'km monitor' to start proxying MCP requests."
                .green()
                .bold()
        );
    }
}

pub struct ProxyService {
    log_repository: LogRepository,
    process_manager: ProcessManager,
}

impl Default for ProxyService {
    fn default() -> Self {
        Self::new()
    }
}

impl ProxyService {
    pub fn new() -> Self {
        Self {
            log_repository: LogRepository::new(),
            process_manager: ProcessManager::new(),
        }
    }

    pub async fn run_proxy(&self, command: ProxyCommand) -> Result<()> {
        self.process_manager
            .run_proxy_process(command, &self.log_repository)
            .await
    }
}
