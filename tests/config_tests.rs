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

#[test]
fn test_config_new() {
    let config = Config::new("test-key".to_string(), "https://test.api.com".to_string());

    assert_eq!(config.api_key, "test-key");
    assert_eq!(config.api_url, "https://test.api.com");
    assert_eq!(config.default_tier, None);
}

#[test]
fn test_config_exists_true() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("exists.json");

    let config = Config::new("key".to_string(), "url".to_string());
    config.save(&config_path).unwrap();

    assert!(Config::exists(&config_path));
}

#[test]
fn test_config_exists_false() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("does_not_exist.json");

    assert!(!Config::exists(&config_path));
}

#[test]
fn test_config_serialization_pretty() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("pretty.json");

    let config = Config {
        api_key: "test-key".to_string(),
        api_url: "https://api.test.com".to_string(),
        default_tier: Some("pro".to_string()),
    };

    config.save(&config_path).unwrap();

    let contents = fs::read_to_string(&config_path).unwrap();
    // Pretty JSON should have newlines
    assert!(contents.contains('\n'));
    assert!(contents.contains("api_key"));
    assert!(contents.contains("api_url"));
    assert!(contents.contains("default_tier"));
}

#[test]
fn test_config_special_characters() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("special.json");

    let config = Config {
        api_key: "key-with-special_chars.123!@#".to_string(),
        api_url: "https://api.test.com:8080/path".to_string(),
        default_tier: Some("tier-1".to_string()),
    };

    config.save(&config_path).unwrap();
    let loaded = Config::load(&config_path).unwrap();

    assert_eq!(loaded.api_key, config.api_key);
    assert_eq!(loaded.api_url, config.api_url);
    assert_eq!(loaded.default_tier, config.default_tier);
}

#[test]
fn test_config_load_with_env_tier_override() {
    let _lock = ENV_TEST_LOCK.lock().unwrap();
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("tier_override.json");

    // Clean up environment
    env::remove_var("KM_API_KEY");
    env::remove_var("KM_API_URL");
    env::remove_var("KM_DEFAULT_TIER");

    // Create config with one tier
    let config = Config {
        api_key: "key".to_string(),
        api_url: "https://api.test.com".to_string(),
        default_tier: Some("free".to_string()),
    };
    config.save(&config_path).unwrap();

    // Override tier with environment variable
    env::set_var("KM_DEFAULT_TIER", "enterprise");

    let loaded = Config::load_with_env(&config_path).unwrap();

    assert_eq!(loaded.default_tier, Some("enterprise".to_string()));

    // Clean up
    env::remove_var("KM_DEFAULT_TIER");
}

#[test]
fn test_config_load_with_partial_env_override() {
    let _lock = ENV_TEST_LOCK.lock().unwrap();
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("partial.json");

    // Clean up environment
    env::remove_var("KM_API_KEY");
    env::remove_var("KM_API_URL");
    env::remove_var("KM_DEFAULT_TIER");

    // Create config
    let config = Config {
        api_key: "file-key".to_string(),
        api_url: "https://file.api.com".to_string(),
        default_tier: Some("basic".to_string()),
    };
    config.save(&config_path).unwrap();

    // Only override API key
    env::set_var("KM_API_KEY", "env-key");

    let loaded = Config::load_with_env(&config_path).unwrap();

    // API key should be from env, others from file
    assert_eq!(loaded.api_key, "env-key");
    assert_eq!(loaded.api_url, "https://file.api.com");
    assert_eq!(loaded.default_tier, Some("basic".to_string()));

    // Clean up
    env::remove_var("KM_API_KEY");
}

#[test]
fn test_config_empty_api_key() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("empty_key.json");

    let config = Config {
        api_key: "".to_string(),
        api_url: "https://api.test.com".to_string(),
        default_tier: None,
    };

    config.save(&config_path).unwrap();
    let loaded = Config::load(&config_path).unwrap();

    assert_eq!(loaded.api_key, "");
}

#[test]
fn test_config_very_long_api_key() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("long_key.json");

    let long_key = "a".repeat(1000);
    let config = Config {
        api_key: long_key.clone(),
        api_url: "https://api.test.com".to_string(),
        default_tier: None,
    };

    config.save(&config_path).unwrap();
    let loaded = Config::load(&config_path).unwrap();

    assert_eq!(loaded.api_key, long_key);
}

#[test]
fn test_config_malformed_json_missing_field() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("malformed.json");

    // Write JSON missing required field
    fs::write(&config_path, r#"{"api_key": "test"}"#).unwrap();

    let result = Config::load(&config_path);
    assert!(result.is_err());
}

#[test]
fn test_config_json_with_extra_fields() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("extra_fields.json");

    // Write JSON with extra fields that should be ignored
    fs::write(
        &config_path,
        r#"{
            "api_key": "test-key",
            "api_url": "https://api.test.com",
            "default_tier": "pro",
            "extra_field": "should_be_ignored",
            "another_field": 123
        }"#,
    )
    .unwrap();

    let result = Config::load(&config_path);
    assert!(result.is_ok());
    let config = result.unwrap();

    assert_eq!(config.api_key, "test-key");
    assert_eq!(config.api_url, "https://api.test.com");
    assert_eq!(config.default_tier, Some("pro".to_string()));
}
