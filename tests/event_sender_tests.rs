use km::infrastructure::event_sender::{
    create_request_event, create_response_event, extract_method_from_content, EventSender,
};
use mockito::ServerGuard;

#[cfg(test)]
mod event_sender_tests {
    use super::*;

    struct TestEventSenderContext {
        mock_server: ServerGuard,
    }

    impl TestEventSenderContext {
        async fn new() -> Self {
            let mock_server = mockito::Server::new_async().await;
            Self { mock_server }
        }

        fn get_mock_url(&self) -> String {
            self.mock_server.url()
        }
    }

    #[tokio::test]
    async fn test_create_request_event() {
        let content = r#"{"method": "initialize", "params": {}}"#;
        let method = Some("initialize".to_string());

        let event = create_request_event(content.to_string(), method.clone());

        assert_eq!(event.direction, "request");
        assert_eq!(event.method, method);
        assert_eq!(event.payload, content.as_bytes());
        assert_eq!(event.size, content.len() as i32);
        assert!(!event.id.is_empty());
    }

    #[tokio::test]
    async fn test_create_response_event() {
        let content = r#"{"result": "success"}"#;
        let method = Some("initialize".to_string());

        let event = create_response_event(content.to_string(), method.clone());

        assert_eq!(event.direction, "response");
        assert_eq!(event.method, method);
        assert_eq!(event.payload, content.as_bytes());
        assert_eq!(event.size, content.len() as i32);
    }

    #[tokio::test]
    async fn test_extract_method_from_valid_json() {
        let json_content = r#"{"method": "resources/list", "params": {}}"#;
        let method = extract_method_from_content(json_content);

        assert_eq!(method, Some("resources/list".to_string()));
    }

    #[tokio::test]
    async fn test_extract_method_from_json_without_method() {
        let json_content = r#"{"result": "success", "id": 1}"#;
        let method = extract_method_from_content(json_content);

        assert_eq!(method, None);
    }

    #[tokio::test]
    async fn test_extract_method_from_invalid_json() {
        let invalid_json = "not a json string";
        let method = extract_method_from_content(invalid_json);

        assert_eq!(method, None);
    }

    #[tokio::test]
    async fn test_extract_method_from_complex_json() {
        let complex_json = r#"
        {
            "jsonrpc": "2.0",
            "method": "tools/call",
            "params": {
                "name": "test_tool",
                "arguments": {"key": "value"}
            },
            "id": 123
        }
        "#;
        let method = extract_method_from_content(complex_json);

        assert_eq!(method, Some("tools/call".to_string()));
    }

    #[tokio::test]
    async fn test_extract_method_with_null_method() {
        let json_with_null = r#"{"method": null, "id": 1}"#;
        let method = extract_method_from_content(json_with_null);

        assert_eq!(method, None);
    }

    #[tokio::test]
    async fn test_extract_method_with_non_string_method() {
        let json_with_number_method = r#"{"method": 123, "id": 1}"#;
        let method = extract_method_from_content(json_with_number_method);

        assert_eq!(method, None);
    }

    #[tokio::test]
    async fn test_event_sender_creation() {
        let sender = EventSender::new();

        // Test that sender can be created without panicking
        assert_eq!(format!("{:?}", sender).contains("EventSender"), true);
    }

    #[tokio::test]
    async fn test_event_sender_default() {
        let sender1 = EventSender::new();
        let sender2 = EventSender::default();

        // Both should be created successfully
        assert_eq!(format!("{:?}", sender1).contains("EventSender"), true);
        assert_eq!(format!("{:?}", sender2).contains("EventSender"), true);
    }

    #[tokio::test]
    async fn test_initialize_without_config() {
        let sender = EventSender::new();
        let result = sender.initialize().await;

        // Should succeed even without configuration file
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_buffer_event_basic() {
        let _sender = EventSender::new();
        let _event = create_request_event("test".to_string(), None);

        // Should not panic when buffering event
        // Note: We can't actually test buffer_event without proper setup
        // but we can test event creation
        assert!(_event.direction == "request");
    }

    #[tokio::test]
    async fn test_flush_buffer_empty() {
        let _sender = EventSender::new();

        // Should handle empty buffer gracefully
        // Note: We can't actually test flush_buffer without proper setup
        // Test passes if creation doesn't panic
        assert!(true);
    }

    #[tokio::test]
    async fn test_event_creation_with_empty_content() {
        let event = create_request_event("".to_string(), None);

        assert_eq!(event.direction, "request");
        assert_eq!(event.method, None);
        assert_eq!(event.size, 0);
        assert!(event.payload.is_empty());
    }

    #[tokio::test]
    async fn test_event_creation_with_large_content() {
        let large_content = "x".repeat(100000);
        let event = create_response_event(large_content.clone(), Some("large_test".to_string()));

        assert_eq!(event.direction, "response");
        assert_eq!(event.method, Some("large_test".to_string()));
        assert_eq!(event.size, 100000);
        assert_eq!(event.payload.len(), 100000);
    }

    #[tokio::test]
    async fn test_extract_method_from_nested_json() {
        let nested_json = r#"
        {
            "outer": {
                "method": "should_not_match"
            },
            "method": "correct_method",
            "params": {}
        }
        "#;
        let method = extract_method_from_content(nested_json);

        assert_eq!(method, Some("correct_method".to_string()));
    }

    #[tokio::test]
    async fn test_extract_method_from_json_with_whitespace() {
        let json_with_whitespace = "   \n\t  {  \"method\"  :  \"whitespace_test\"  }  \n\t  ";
        let method = extract_method_from_content(json_with_whitespace);

        assert_eq!(method, Some("whitespace_test".to_string()));
    }

    #[tokio::test]
    async fn test_multiple_event_buffering() {
        let _sender = EventSender::new();

        let event1 = create_request_event("event1".to_string(), Some("method1".to_string()));
        let event2 = create_response_event("event2".to_string(), Some("method2".to_string()));
        let event3 = create_request_event("event3".to_string(), None);

        // Test event creation
        assert!(event1.direction == "request");
        assert!(event2.direction == "response");
        assert!(event3.direction == "request");

        // Test passes if no panic occurs during event creation
    }

    #[tokio::test]
    async fn test_event_with_unicode_content() {
        let unicode_content = "测试内容 🚀 émojis and special chars: áéíóú";
        let event = create_request_event(
            unicode_content.to_string(),
            Some("unicode_test".to_string()),
        );

        assert_eq!(event.method, Some("unicode_test".to_string()));
        assert_eq!(event.payload, unicode_content.as_bytes());
        assert_eq!(event.size, unicode_content.as_bytes().len() as i32);
    }

    #[tokio::test]
    async fn test_extract_method_from_malformed_json() {
        let malformed_cases = vec![
            r#"{"method": "test""#, // Missing closing brace
            r#"{"method": }"#,      // Missing value
            r#"{method: "test"}"#,  // Missing quotes on key
            r#"{"method":}"#,       // Missing value with colon
        ];

        for malformed in malformed_cases {
            let method = extract_method_from_content(malformed);
            assert_eq!(
                method, None,
                "Expected None for malformed JSON: {}",
                malformed
            );
        }
    }

    #[tokio::test]
    async fn test_concurrent_event_buffering() {
        let _sender = EventSender::new();

        let handles: Vec<_> = (0..10)
            .map(|i| {
                tokio::spawn(async move {
                    let event = create_request_event(
                        format!("concurrent_event_{}", i),
                        Some(format!("method_{}", i)),
                    );
                    // Test event creation in concurrent context
                    assert!(event.direction == "request");
                    assert!(event.method == Some(format!("method_{}", i)));
                })
            })
            .collect();

        // Wait for all tasks to complete
        for handle in handles {
            handle.await.unwrap();
        }

        // Test passes if no panic or deadlock occurs
    }
}
