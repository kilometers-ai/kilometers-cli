# IDE Setup Guide

## GoLand/IntelliJ IDEA Configuration

This project includes shared GoLand/IntelliJ IDEA run configurations to help developers quickly get started with running and debugging the application.

### Included IDE Files

The following `.idea` files are committed to version control as they contain useful shared configurations:

- `.idea/runConfigurations/` - Shared run configurations for common tasks:
  - `debug_monitor_command.xml` - Debug the monitor command
  - `km_test_*.xml` - Various test configurations
  - `km_plugins_install_with_env.xml` - Install plugins with environment variables
  - `km_dev_plugin_test_api_logger.xml` - Test API logger plugin

- `.idea/vcs.xml` - Version control system mappings
- `.idea/encodings.xml` - File encoding settings

### Excluded IDE Files

The following `.idea` files are ignored as they contain user-specific or system-specific settings:

- `.idea/workspace.xml` - User workspace settings
- `.idea/*.iml` - Module files with potential absolute paths
- `.idea/modules.xml` - Module configuration with absolute paths
- `.idea/dataSources/` - Database connection details
- `.idea/shelf/` - Local shelved changes
- `.idea/httpRequests/` - HTTP client request history
- Other user-specific files (see `.gitignore` for complete list)

### Using Run Configurations

1. Open the project in GoLand/IntelliJ IDEA
2. The run configurations will automatically be available in the Run/Debug dropdown
3. Select a configuration and click Run or Debug

### Adding New Run Configurations

When adding new run configurations that would be useful for other developers:

1. Create the configuration in GoLand
2. Ensure it uses project-relative paths (using `$PROJECT_DIR$`)
3. Use environment variable placeholders where appropriate (e.g., `$USER_HOME$`)
4. Save the configuration as a "Project" configuration (not "Local")
5. The configuration will be saved in `.idea/runConfigurations/`
6. Commit the new configuration file

### Environment Variables

Common environment variables used in run configurations:

- `KM_API_KEY` - API key for Kilometers service
- `KM_API_ENDPOINT` - API endpoint URL
- `KM_DEBUG` - Enable debug mode
- `KM_LOG_LEVEL` - Log level (debug, info, warn, error)
- `KM_PLUGINS_DIR` - Directory containing plugins

Note: Sensitive values should be set locally and not committed. Use placeholder values in shared configurations.