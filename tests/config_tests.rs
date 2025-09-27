use km::config::Config;
use std::env;
use std::fs;
use std::sync::Mutex;
use tempfile::TempDir;

// Mutex to serialize environment variable tests to prevent contamination
static ENV_TEST_LOCK: Mutex<()> = Mutex::new(());

#[test]
fn test_config_creation() {
    let config = Config {
        api_key: "test-api-key".to_string(),
        api_url: "https://api.kilometers.ai".to_string(),
        default_tier: None,
    };
    assert_eq!(config.api_key, "test-api-key");
    assert_eq!(config.api_url, "https://api.kilometers.ai");
    assert_eq!(config.default_tier, None);
}

#[test]
fn test_config_save_and_load() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("test_config.json");

    let original_config = Config {
        api_key: "save-test-key".to_string(),
        api_url: "https://test.api.com".to_string(),
        default_tier: Some("pro".to_string()),
    };

    original_config.save(&config_path).unwrap();

    let loaded_config = Config::load(&config_path).unwrap();
    assert_eq!(loaded_config.api_key, original_config.api_key);
    assert_eq!(loaded_config.api_url, original_config.api_url);
    assert_eq!(loaded_config.default_tier, original_config.default_tier);
}

#[test]
fn test_config_load_nonexistent_file() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("nonexistent.json");

    let result = Config::load(&config_path);
    assert!(result.is_err());
}

#[test]
fn test_config_load_invalid_json() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("invalid.json");

    fs::write(&config_path, "{ invalid json }").unwrap();

    let result = Config::load(&config_path);
    assert!(result.is_err());
}

#[test]
fn test_config_with_default_tier_none() {
    let config = Config {
        api_key: "test-key".to_string(),
        api_url: "https://api.test.com".to_string(),
        default_tier: None,
    };

    let json = serde_json::to_string(&config).unwrap();
    assert!(!json.contains("default_tier"));
}

#[test]
fn test_config_load_with_env_file_exists() {
    let _lock = ENV_TEST_LOCK.lock().unwrap();
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("test_config.json");

    // Clean up any existing environment variables first
    env::remove_var("KM_API_KEY");
    env::remove_var("KM_API_URL");
    env::remove_var("KM_DEFAULT_TIER");

    // Create a config file
    let original_config = Config {
        api_key: "file-key".to_string(),
        api_url: "https://file.api.com".to_string(),
        default_tier: Some("basic".to_string()),
    };
    original_config.save(&config_path).unwrap();

    // Set environment variables to override
    env::set_var("KM_API_KEY", "env-key");
    env::set_var("KM_API_URL", "https://env.api.com");

    let loaded_config = Config::load_with_env(&config_path).unwrap();

    // Environment variables should override file values
    assert_eq!(loaded_config.api_key, "env-key");
    assert_eq!(loaded_config.api_url, "https://env.api.com");
    assert_eq!(loaded_config.default_tier, Some("basic".to_string()));

    // Clean up
    env::remove_var("KM_API_KEY");
    env::remove_var("KM_API_URL");
}

#[test]
fn test_config_load_with_env_no_file() {
    let _lock = ENV_TEST_LOCK.lock().unwrap();
    let temp_dir = TempDir::new().unwrap();
    let nonexistent_path = temp_dir.path().join("nonexistent.json");

    // Clean up any existing environment variables first
    env::remove_var("KM_API_KEY");
    env::remove_var("KM_API_URL");
    env::remove_var("KM_DEFAULT_TIER");

    // Set required environment variables
    env::set_var("KM_API_KEY", "env-only-key");
    env::set_var("KM_API_URL", "https://env-only.api.com");
    env::set_var("KM_DEFAULT_TIER", "premium");

    let loaded_config = Config::load_with_env(&nonexistent_path).unwrap();

    // Should create config from environment variables
    assert_eq!(loaded_config.api_key, "env-only-key");
    assert_eq!(loaded_config.api_url, "https://env-only.api.com");
    assert_eq!(loaded_config.default_tier, Some("premium".to_string()));

    // Clean up
    env::remove_var("KM_API_KEY");
    env::remove_var("KM_API_URL");
    env::remove_var("KM_DEFAULT_TIER");
}

#[test]
fn test_config_load_with_env_no_file_missing_api_key() {
    let _lock = ENV_TEST_LOCK.lock().unwrap();
    let temp_dir = TempDir::new().unwrap();
    let nonexistent_path = temp_dir.path().join("nonexistent.json");

    // Clean up any existing environment variables first
    env::remove_var("KM_API_KEY");
    env::remove_var("KM_API_URL");
    env::remove_var("KM_DEFAULT_TIER");

    let result = Config::load_with_env(&nonexistent_path);
    assert!(result.is_err());
    assert!(result
        .unwrap_err()
        .to_string()
        .contains("KM_API_KEY not set"));
}

#[test]
fn test_config_load_with_env_defaults() {
    let _lock = ENV_TEST_LOCK.lock().unwrap();
    let temp_dir = TempDir::new().unwrap();
    let nonexistent_path = temp_dir.path().join("nonexistent.json");

    // Clean up any existing environment variables first
    env::remove_var("KM_API_KEY");
    env::remove_var("KM_API_URL");
    env::remove_var("KM_DEFAULT_TIER");

    // Set only the required API key
    env::set_var("KM_API_KEY", "minimal-key");

    let loaded_config = Config::load_with_env(&nonexistent_path).unwrap();

    // Should use defaults for missing values
    assert_eq!(loaded_config.api_key, "minimal-key");
    assert_eq!(loaded_config.api_url, "https://api.kilometers.ai");
    assert_eq!(loaded_config.default_tier, None);

    // Clean up
    env::remove_var("KM_API_KEY");
}
