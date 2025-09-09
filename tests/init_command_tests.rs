use km::domain::auth::{ApiKeyCredentials, Configuration};
use km::infrastructure::api_client::ApiClient;
use mockito::{Matcher, ServerGuard};
use std::fs;
use std::path::PathBuf;
use tempfile::TempDir;

#[cfg(test)]
mod init_command_integration_tests {
    use super::*;

    struct TestConfigurationRepository {
        config_path: PathBuf,
    }

    impl TestConfigurationRepository {
        fn with_path(path: PathBuf) -> Self {
            Self { config_path: path }
        }

        fn save_configuration(&self, config: Configuration) -> anyhow::Result<()> {
            let config_content = format!(r#"{{"api_key": "{}"}}"#, config.api_key);
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
    }

    struct TestContext {
        _temp_dir: TempDir, // prefixed with _ to indicate intentionally unused
        config_path: PathBuf,
        mock_server: ServerGuard,
    }

    impl TestContext {
        async fn new() -> Self {
            let temp_dir = TempDir::new().unwrap();
            let config_path = temp_dir.path().join("km_config.json");
            let mock_server = mockito::Server::new_async().await;

            Self {
                _temp_dir: temp_dir,
                config_path,
                mock_server,
            }
        }

        fn get_test_api_client(&self) -> ApiClient {
            ApiClient::new(self.mock_server.url())
        }

        fn get_test_config_repository(&self) -> TestConfigurationRepository {
            TestConfigurationRepository::with_path(self.config_path.clone())
        }

        fn create_test_auth_response(success: bool) -> String {
            if success {
                r#"{
                    "success": true,
                    "customer": {
                        "id": "test-user-123",
                        "email": "test@example.com",
                        "organization": "Test Org",
                        "subscriptionPlan": "Pro",
                        "subscriptionStatus": "Active",
                        "hasPassword": true,
                        "lastLoginAt": "2024-01-01T00:00:00Z",
                        "createdAt": "2023-01-01T00:00:00Z"
                    },
                    "token": {
                        "accessToken": "test-access-token",
                        "refreshToken": "test-refresh-token",
                        "accessTokenExpiresAt": "2024-01-02T00:00:00Z",
                        "refreshTokenExpiresAt": "2024-02-01T00:00:00Z"
                    }
                }"#
                .to_string()
            } else {
                r#"{
                    "success": false,
                    "customer": {
                        "id": "",
                        "email": "",
                        "subscriptionPlan": "",
                        "subscriptionStatus": "",
                        "hasPassword": false,
                        "createdAt": ""
                    },
                    "token": {
                        "accessToken": "",
                        "refreshToken": "",
                        "accessTokenExpiresAt": "",
                        "refreshTokenExpiresAt": ""
                    }
                }"#
                .to_string()
            }
        }
    }

    #[tokio::test]
    async fn test_successful_initialization_with_valid_api_key() {
        let mut context = TestContext::new().await;
        let test_api_key = "test-api-key-123";

        let _mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": test_api_key
            })))
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(TestContext::create_test_auth_response(true))
            .create_async()
            .await;

        let api_client = context.get_test_api_client();
        let config_repo = context.get_test_config_repository();

        let credentials = ApiKeyCredentials::new(test_api_key.to_string());
        let result = api_client.authenticate(credentials).await.unwrap();

        assert!(result.success);
        assert_eq!(result.customer.email, "test@example.com");
        assert_eq!(result.customer.subscription_plan, "Pro");

        config_repo
            .save_configuration(Configuration::new(test_api_key.to_string()))
            .unwrap();

        let saved_config = config_repo.load_configuration().unwrap();
        assert_eq!(saved_config.api_key, test_api_key);

        assert!(context.config_path.exists());
    }

    #[tokio::test]
    async fn test_initialization_fails_with_invalid_api_key() {
        let mut context = TestContext::new().await;
        let invalid_api_key = "invalid-key";

        let _mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": invalid_api_key
            })))
            .with_status(401)
            .with_header("content-type", "application/json")
            .with_body(TestContext::create_test_auth_response(false))
            .create_async()
            .await;

        let api_client = context.get_test_api_client();
        let credentials = ApiKeyCredentials::new(invalid_api_key.to_string());
        let result = api_client.authenticate(credentials).await;

        assert!(result.is_err());
        match result {
            Err(km::domain::auth::AuthenticationError::InvalidApiKey) => {}
            _ => panic!("Expected InvalidApiKey error"),
        }

        assert!(!context.config_path.exists());
    }

    #[tokio::test]
    async fn test_initialization_handles_empty_api_key() {
        let context = TestContext::new().await;
        let empty_api_key = "";

        let config_repo = context.get_test_config_repository();

        let result = Configuration::new(empty_api_key.to_string());
        assert!(result.api_key.is_empty());

        config_repo.save_configuration(result.clone()).unwrap();
        let loaded = config_repo.load_configuration().unwrap();
        assert!(loaded.api_key.is_empty());
    }

    #[tokio::test]
    async fn test_initialization_handles_network_error() {
        let mut context = TestContext::new().await;
        let test_api_key = "test-key";

        let _mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": test_api_key
            })))
            .with_status(500)
            .with_body("Internal Server Error")
            .create_async()
            .await;

        let api_client = context.get_test_api_client();
        let credentials = ApiKeyCredentials::new(test_api_key.to_string());
        let result = api_client.authenticate(credentials).await;

        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_configuration_file_persistence() {
        let context = TestContext::new().await;
        let test_api_key = "persistent-test-key";

        let config_repo = context.get_test_config_repository();

        let original_config = Configuration::new(test_api_key.to_string());
        config_repo
            .save_configuration(original_config.clone())
            .unwrap();

        assert!(context.config_path.exists());

        let file_content = fs::read_to_string(&context.config_path).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&file_content).unwrap();
        assert_eq!(parsed["api_key"], test_api_key);

        let loaded_config = config_repo.load_configuration().unwrap();
        assert_eq!(loaded_config.api_key, test_api_key);

        let another_repo = TestConfigurationRepository::with_path(context.config_path.clone());
        let reloaded_config = another_repo.load_configuration().unwrap();
        assert_eq!(reloaded_config.api_key, test_api_key);
    }

    #[tokio::test]
    async fn test_configuration_overwrites_existing_file() {
        let context = TestContext::new().await;
        let config_repo = context.get_test_config_repository();

        let first_config = Configuration::new("first-key".to_string());
        config_repo.save_configuration(first_config).unwrap();

        let second_config = Configuration::new("second-key".to_string());
        config_repo.save_configuration(second_config).unwrap();

        let loaded_config = config_repo.load_configuration().unwrap();
        assert_eq!(loaded_config.api_key, "second-key");

        let file_content = fs::read_to_string(&context.config_path).unwrap();
        assert!(!file_content.contains("first-key"));
    }

    #[tokio::test]
    async fn test_api_response_parsing_with_all_fields() {
        let mut context = TestContext::new().await;
        let test_api_key = "full-response-key";

        let _mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": test_api_key
            })))
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(TestContext::create_test_auth_response(true))
            .create_async()
            .await;

        let api_client = context.get_test_api_client();
        let credentials = ApiKeyCredentials::new(test_api_key.to_string());
        let result = api_client.authenticate(credentials).await.unwrap();

        assert!(result.success);
        assert_eq!(result.customer.id, "test-user-123");
        assert_eq!(result.customer.email, "test@example.com");
        assert_eq!(result.customer.organization, Some("Test Org".to_string()));
        assert_eq!(result.customer.subscription_plan, "Pro");
        assert_eq!(result.customer.subscription_status, "Active");
        assert!(result.customer.has_password);
        assert_eq!(
            result.customer.last_login_at,
            Some("2024-01-01T00:00:00Z".to_string())
        );
        assert_eq!(result.customer.created_at, "2023-01-01T00:00:00Z");
    }

    #[tokio::test]
    async fn test_api_response_parsing_with_minimal_fields() {
        let mut context = TestContext::new().await;
        let test_api_key = "minimal-response-key";

        let minimal_response = r#"{
            "success": true,
            "customer": {
                "id": "minimal-user",
                "email": "minimal@example.com",
                "subscriptionPlan": "Free",
                "subscriptionStatus": "Active",
                "hasPassword": false,
                "createdAt": "2024-01-01T00:00:00Z"
            },
            "token": {
                "accessToken": "test-access-token",
                "refreshToken": "test-refresh-token",
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
            .with_body(minimal_response)
            .create_async()
            .await;

        let api_client = context.get_test_api_client();
        let credentials = ApiKeyCredentials::new(test_api_key.to_string());
        let result = api_client.authenticate(credentials).await.unwrap();

        assert!(result.success);
        assert_eq!(result.customer.id, "minimal-user");
        assert_eq!(result.customer.email, "minimal@example.com");
        assert_eq!(result.customer.organization, None);
        assert_eq!(result.customer.subscription_plan, "Free");
        assert!(!result.customer.has_password);
        assert_eq!(result.customer.last_login_at, None);
    }

    #[tokio::test]
    async fn test_configuration_handles_special_characters_in_api_key() {
        let context = TestContext::new().await;
        let special_api_key = "test-key!@#$%^&*()_+-=[]{}|;:',.<>?/~`";

        let config_repo = context.get_test_config_repository();
        let config = Configuration::new(special_api_key.to_string());
        config_repo.save_configuration(config).unwrap();

        let loaded_config = config_repo.load_configuration().unwrap();
        assert_eq!(loaded_config.api_key, special_api_key);
    }

    #[tokio::test]
    async fn test_initialization_with_timeout_handling() {
        let mut context = TestContext::new().await;
        let test_api_key = "timeout-test-key";

        let _mock = context
            .mock_server
            .mock("POST", "/api/auth/token")
            .match_body(Matcher::Json(serde_json::json!({
                "apiKey": test_api_key
            })))
            .with_status(408)
            .with_body("Request Timeout")
            .create_async()
            .await;

        let api_client = context.get_test_api_client();
        let credentials = ApiKeyCredentials::new(test_api_key.to_string());
        let result = api_client.authenticate(credentials).await;

        assert!(result.is_err());
    }
}
