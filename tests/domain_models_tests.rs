use chrono::Utc;
use km::domain::proxy::{McpEvent, McpEventBatch, ProxyCommand};
use uuid::Uuid;

#[cfg(test)]
mod domain_tests {
    use super::*;

    #[test]
    fn test_proxy_command_creation() {
        let command = ProxyCommand::new(
            "node".to_string(),
            vec![
                "server.js".to_string(),
                "--port".to_string(),
                "3000".to_string(),
            ],
        );

        assert_eq!(command.command, "node");
        assert_eq!(command.args.len(), 3);
        assert_eq!(command.args[0], "server.js");
        assert_eq!(command.args[1], "--port");
        assert_eq!(command.args[2], "3000");
    }

    #[test]
    fn test_proxy_command_empty_args() {
        let command = ProxyCommand::new("ls".to_string(), vec![]);

        assert_eq!(command.command, "ls");
        assert!(command.args.is_empty());
    }

    #[test]
    fn test_proxy_command_clone() {
        let command1 = ProxyCommand::new("echo".to_string(), vec!["hello".to_string()]);
        let command2 = command1.clone();

        assert_eq!(command1.command, command2.command);
        assert_eq!(command1.args, command2.args);
    }

    #[test]
    fn test_proxy_command_debug() {
        let command = ProxyCommand::new("test".to_string(), vec!["arg1".to_string()]);
        let debug_str = format!("{:?}", command);

        assert!(debug_str.contains("ProxyCommand"));
        assert!(debug_str.contains("test"));
        assert!(debug_str.contains("arg1"));
    }

    #[test]
    fn test_mcp_event_new_request() {
        let content = r#"{"method": "initialize", "params": {}}"#;
        let method = Some("initialize".to_string());
        let correlation_id = Some(Uuid::new_v4());

        let event = McpEvent::new_request(content.to_string(), method.clone(), correlation_id);

        assert!(!event.id.is_empty());
        assert_eq!(event.direction, "request");
        assert_eq!(event.method, method);
        assert!(event.correlation_id.is_some());
        assert_eq!(event.size, content.len() as i32);
        assert_eq!(event.payload, content.as_bytes());
        assert!((Utc::now() - event.timestamp).num_seconds() < 2);
    }

    #[test]
    fn test_mcp_event_new_response() {
        let content = r#"{"result": "success"}"#;
        let method = Some("initialize".to_string());
        let correlation_id = Some(Uuid::new_v4());

        let event = McpEvent::new_response(content.to_string(), method.clone(), correlation_id);

        assert!(!event.id.is_empty());
        assert_eq!(event.direction, "response");
        assert_eq!(event.method, method);
        assert!(event.correlation_id.is_some());
        assert_eq!(event.size, content.len() as i32);
        assert_eq!(event.payload, content.as_bytes());
    }

    #[test]
    fn test_mcp_event_without_method() {
        let content = "test content".to_string();
        let event = McpEvent::new_request(content.clone(), None, None);

        assert_eq!(event.method, None);
        assert_eq!(event.correlation_id, None);
        assert_eq!(event.payload, content.as_bytes());
    }

    #[test]
    fn test_mcp_event_serialization() {
        let content = r#"{"test": "data"}"#;
        let event = McpEvent::new_request(content.to_string(), Some("test".to_string()), None);

        // Test that serialization includes base64 encoding
        let json = serde_json::to_string(&event).unwrap();
        assert!(json.contains("request"));
        assert!(json.contains("test"));
        // Should contain base64 encoded payload
        assert!(json.contains("payload"));
    }

    #[test]
    fn test_mcp_event_batch_creation() {
        let event1 =
            McpEvent::new_request("content1".to_string(), Some("method1".to_string()), None);
        let event2 =
            McpEvent::new_response("content2".to_string(), Some("method2".to_string()), None);

        let batch = McpEventBatch {
            events: vec![event1, event2],
            cli_version: "0.2.0".to_string(),
            batch_timestamp: Utc::now(),
            correlation_id: Some("batch-123".to_string()),
        };

        assert_eq!(batch.events.len(), 2);
        assert_eq!(batch.cli_version, "0.2.0");
        assert!(batch.correlation_id.is_some());
    }

    #[test]
    fn test_mcp_event_batch_serialization() {
        let event = McpEvent::new_request("test".to_string(), None, None);
        let batch = McpEventBatch {
            events: vec![event],
            cli_version: "0.2.0".to_string(),
            batch_timestamp: Utc::now(),
            correlation_id: None,
        };

        let json = serde_json::to_string(&batch).unwrap();
        assert!(json.contains("cliVersion"));
        assert!(json.contains("batchTimestamp"));
        assert!(json.contains("events"));
    }

    #[test]
    fn test_mcp_event_with_large_payload() {
        let large_content = "x".repeat(10000);
        let event = McpEvent::new_request(large_content.clone(), None, None);

        assert_eq!(event.size, 10000);
        assert_eq!(event.payload.len(), 10000);
    }

    #[test]
    fn test_mcp_event_clone() {
        let event1 = McpEvent::new_request("content".to_string(), Some("method".to_string()), None);
        let event2 = event1.clone();

        assert_eq!(event1.id, event2.id);
        assert_eq!(event1.direction, event2.direction);
        assert_eq!(event1.method, event2.method);
        assert_eq!(event1.payload, event2.payload);
    }

    #[test]
    fn test_mcp_event_debug() {
        let event =
            McpEvent::new_request("debug test".to_string(), Some("debug".to_string()), None);
        let debug_str = format!("{:?}", event);

        assert!(debug_str.contains("McpEvent"));
        assert!(debug_str.contains("request"));
        assert!(debug_str.contains("debug"));
    }

    #[test]
    fn test_empty_content_handling() {
        let event = McpEvent::new_request("".to_string(), None, None);

        assert_eq!(event.size, 0);
        assert!(event.payload.is_empty());
        assert!(!event.id.is_empty()); // ID should still be generated
    }

    #[test]
    fn test_unicode_content() {
        let unicode_content = "测试内容 🚀 émojis";
        let event = McpEvent::new_request(unicode_content.to_string(), None, None);

        assert_eq!(event.payload, unicode_content.as_bytes());
        assert_eq!(event.size, unicode_content.as_bytes().len() as i32);
    }

    #[test]
    fn test_correlation_id_string_conversion() {
        let uuid = Uuid::new_v4();
        let event = McpEvent::new_request("test".to_string(), None, Some(uuid));

        assert_eq!(event.correlation_id, Some(uuid.to_string()));
    }
}
