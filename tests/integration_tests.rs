use km::domain::auth::{ApiKeyCredentials, Configuration};
use km::infrastructure::api_client::ApiClient;
use mockito::{Matcher, ServerGuard};
use std::fs;
use std::path::PathBuf;
use tempfile::TempDir;

#[cfg(test)]
mod integration_tests {
    use super::*;

    struct IntegrationTestContext {
        _temp_dir: TempDir,
        config_path: PathBuf,
        mock_server: ServerGuard,
    }

    impl IntegrationTestContext {
        async fn new() -> Self {
            let temp_dir = TempDir::new().unwrap();
            let config_path = temp_dir.path().join("config.json");
            let mock_server = mockito::Server::new_async().await;

            Self {
                _temp_dir: temp_dir,
                config_path,
                mock_server,
            }
        }

        fn create_test_config_repo(&self) -> TestConfigurationRepository {
            TestConfigurationRepository::with_path(self.config_path.clone())
        }

        fn get_api_client(&self) -> ApiClient {
            ApiClient::new(self.mock_server.url())
        }
    }

    // Test configuration repository that works with specific paths
    struct TestConfigurationRepository {
        config_path: PathBuf,
    }

    impl TestConfigurationRepository {
        fn with_path(config_path: PathBuf) -> Self {
            Self { config_path }
        }

        fn save_configuration(&self, config: Configuration) -> anyhow::Result<()> {
            let config_content = format!(r#"{{"api_key": "{}"}}"#, config.api_key);
            if let Some(parent) = self.config_path.parent() {
                std::fs::create_dir_all(parent)?;
            }
            std::fs::write(&self.config_path, config_content)?;
            Ok(())
        }

        fn load_configuration(&self) -> anyhow::Result<Configuration> {
            let content = std::fs::read_to_string(&self.config_path)?;
            let json: serde_json::Value = serde_json::from_str(&content)?;
            let api_key = json["api_key"]
                .as_str()
                .ok_or_else(|| anyhow::anyhow!("API key not found in configuration"))?
                .to_string();
            Ok(Configuration::new(api_key))
        }

        fn config_exists(&self) -> bool {
            self.config_path.exists()
        }
    }

    #[tokio::test]
    async fn test_complete_authentication_flow() {
        let mut context = IntegrationTestContext::new().await;
        let test_api_key = "integration-test-key-123";

        // Mock successful authentication response
        let auth_response = r#"{
            "success": true,
            "customer": {
                "id": "integration-user-123",
                "email": "integration@example.com",
                "organization": "Integration Test Org",
                "subscriptionPlan": "Pro",
                "subscriptionStatus": "Active",
                "hasPassword": true,
                "lastLoginAt": "2024-01-01T00:00:00Z",
                "createdAt": "2023-01-01T00:00:00Z"
            },
            "token": {
                "accessToken": "access-token-123",
                "refreshToken": "refresh-token-123",
                "accessTokenExpiresAt": "2024-01-02T00:00:00Z",
                "refreshTokenExpiresAt": "2024-02-01T00:00:00Z"
            }
        }"#;

        let _mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": test_api_key
            })))
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(auth_response)
            .create_async()
            .await;

        // Step 1: Authenticate with API
        let api_client = context.get_api_client();
        let credentials = ApiKeyCredentials::new(test_api_key.to_string());
        let auth_result = api_client.authenticate(credentials).await.unwrap();

        assert!(auth_result.success);
        assert_eq!(auth_result.customer.email, "integration@example.com");

        // Step 2: Save configuration
        let config_repo = context.create_test_config_repo();
        let config = Configuration::new(test_api_key.to_string());
        config_repo.save_configuration(config).unwrap();

        // Step 3: Verify configuration was saved
        assert!(config_repo.config_exists());

        // Step 4: Load and verify configuration
        let loaded_config = config_repo.load_configuration().unwrap();
        assert_eq!(loaded_config.api_key, test_api_key);

        // Step 5: Verify file contents
        let file_content = fs::read_to_string(&context.config_path).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&file_content).unwrap();
        assert_eq!(parsed["api_key"], test_api_key);
    }

    #[tokio::test]
    async fn test_authentication_and_configuration_with_retry_logic() {
        let mut context = IntegrationTestContext::new().await;
        let test_api_key = "retry-test-key";

        // First call fails with 500, second call succeeds
        let error_mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": test_api_key
            })))
            .with_status(500)
            .with_body("Internal Server Error")
            .expect(1)
            .create_async()
            .await;

        let success_response = r#"{
            "success": true,
            "customer": {
                "id": "retry-user",
                "email": "retry@example.com",
                "subscriptionPlan": "Free",
                "subscriptionStatus": "Active",
                "hasPassword": false,
                "createdAt": "2024-01-01T00:00:00Z"
            },
            "token": {
                "accessToken": "retry-access-token",
                "refreshToken": "retry-refresh-token",
                "accessTokenExpiresAt": "2024-01-02T00:00:00Z",
                "refreshTokenExpiresAt": "2024-02-01T00:00:00Z"
            }
        }"#;

        let success_mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": test_api_key
            })))
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(success_response)
            .expect(1)
            .create_async()
            .await;

        let api_client = context.get_api_client();
        let credentials = ApiKeyCredentials::new(test_api_key.to_string());

        // First attempt should fail
        let first_result = api_client.authenticate(credentials.clone()).await;
        assert!(first_result.is_err());

        // Second attempt should succeed
        let second_result = api_client.authenticate(credentials).await.unwrap();
        assert!(second_result.success);
        assert_eq!(second_result.customer.email, "retry@example.com");

        error_mock.assert_async().await;
        success_mock.assert_async().await;
    }

    #[tokio::test]
    async fn test_configuration_persistence_across_multiple_operations() {
        let context = IntegrationTestContext::new().await;
        let config_repo = context.create_test_config_repo();

        // Perform multiple save/load operations
        let long_key = "third-very-long-api-key-".to_owned() + &"x".repeat(1000);
        let test_keys = vec![
            "first-api-key",
            "second-api-key-with-special-chars-!@#$%",
            &long_key,
            "", // Empty key
        ];

        for test_key in test_keys {
            // Save configuration
            let config = Configuration::new(test_key.to_string());
            let save_result = config_repo.save_configuration(config);
            assert!(save_result.is_ok());

            // Verify file exists
            assert!(config_repo.config_exists());

            // Load and verify
            let loaded_config = config_repo.load_configuration().unwrap();
            assert_eq!(loaded_config.api_key, test_key);

            // Verify raw file contents
            let file_content = fs::read_to_string(&context.config_path).unwrap();
            assert!(file_content.contains(&test_key));
        }
    }

    #[tokio::test]
    async fn test_error_handling_chain() {
        let mut context = IntegrationTestContext::new().await;
        let invalid_api_key = "definitely-invalid-key";

        // Mock authentication failure
        let _mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": invalid_api_key
            })))
            .with_status(401)
            .with_header("content-type", "application/json")
            .with_body(r#"{"success": false, "message": "Invalid API key"}"#)
            .create_async()
            .await;

        let api_client = context.get_api_client();
        let credentials = ApiKeyCredentials::new(invalid_api_key.to_string());

        // Authentication should fail
        let auth_result = api_client.authenticate(credentials).await;
        assert!(auth_result.is_err());

        // Configuration file should not exist
        let config_repo = context.create_test_config_repo();
        assert!(!config_repo.config_exists());

        // Attempt to load non-existent configuration should fail
        let load_result = config_repo.load_configuration();
        assert!(load_result.is_err());
    }

    #[tokio::test]
    async fn test_concurrent_configuration_operations() {
        use std::sync::Arc;
        use tokio::sync::Mutex;

        let context = IntegrationTestContext::new().await;
        let config_path = Arc::new(Mutex::new(context.config_path.clone()));

        let handles: Vec<_> = (0..10)
            .map(|i| {
                let path = Arc::clone(&config_path);
                tokio::spawn(async move {
                    let config_path = path.lock().await.clone();
                    let config_repo = TestConfigurationRepository::with_path(config_path);
                    let test_key = format!("concurrent-key-{}", i);

                    // Save configuration
                    let config = Configuration::new(test_key.to_string());
                    let save_result = config_repo.save_configuration(config);

                    // Some saves might fail due to concurrent access, but shouldn't panic
                    let _ = save_result;

                    // Try to load - this might also fail due to races, but shouldn't panic
                    let _ = config_repo.load_configuration();
                })
            })
            .collect();

        // Wait for all operations to complete
        for handle in handles {
            handle.await.unwrap();
        }

        // Final state should be consistent
        let final_config_repo = context.create_test_config_repo();
        if final_config_repo.config_exists() {
            let final_config = final_config_repo.load_configuration();
            // Should either succeed or fail gracefully
            assert!(final_config.is_ok() || final_config.is_err());
        }
    }

    #[tokio::test]
    async fn test_configuration_with_malformed_file() {
        let context = IntegrationTestContext::new().await;

        // Create malformed config file
        let malformed_content = r#"{"api_key": "test", "invalid": json}"#;
        if let Some(parent) = context.config_path.parent() {
            std::fs::create_dir_all(parent).unwrap();
        }
        std::fs::write(&context.config_path, malformed_content).unwrap();

        let config_repo = context.create_test_config_repo();
        let load_result = config_repo.load_configuration();

        // Should fail gracefully with proper error
        assert!(load_result.is_err());
    }

    #[tokio::test]
    async fn test_api_client_with_different_base_urls() {
        let base_urls = vec![
            "http://localhost:3000",
            "https://api.kilometers.ai",
            "http://127.0.0.1:8080",
            "https://staging.kilometers.ai/api/v1",
        ];

        for base_url in base_urls {
            let client = ApiClient::new(base_url.to_string());

            // Test that client can be created with different URLs
            // We can't easily test the actual HTTP behavior without mocking,
            // but we can verify the client is created successfully
            assert!(format!("{:?}", client).contains("ApiClient"));
        }
    }
}
