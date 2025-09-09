// `use` statements import types and functions from other modules
// `anyhow::Result` is a type alias for Result<T, anyhow::Error> - provides easy error handling
use anyhow::Result;
// Import I/O traits and types from Rust's standard library
use std::io::{self, BufRead, BufReader, Write};
use std::process::Stdio;
// Import async versions of I/O operations from the tokio runtime
// `AsyncBufReadExt` and `AsyncWriteExt` are traits that add async methods to types
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader as AsyncBufReader};
// Import and rename AsyncCommand to avoid confusion with std::process::Command
use tokio::process::Command as AsyncCommand;
// `crate::` refers to the root of our current project
use crate::domain::proxy::{LogEntry, ProxyCommand};
use crate::infrastructure::event_sender::{
    create_request_event, create_response_event, extract_method_from_content, EventSender,
};
use crate::infrastructure::log_repository::LogRepository;

// `pub` makes this struct public (accessible from other modules)
// Unit struct - has no fields, used for grouping related functions
pub struct ProcessManager;

// `impl` block defines methods for the ProcessManager struct
impl ProcessManager {
    // Associated function (like a static method in other languages)
    // `Self` is an alias for the type we're implementing (ProcessManager)
    pub fn new() -> Self {
        // Return an instance of ProcessManager (empty struct)
        Self
    }

    // `async` function - can use await and runs asynchronously
    // `&self` - immutable reference to self (like 'this' in other languages)
    // `&LogRepository` - borrows LogRepository immutably (doesn't take ownership)
    // `Result<()>` - Returns either Ok(()) for success or Err for failure
    pub async fn run_proxy_process(
        &self,
        command: ProxyCommand,
        log_repo: &LogRepository,
    ) -> Result<()> {
        // Create and initialize event sender
        let event_sender = EventSender::new();
        if let Err(e) = event_sender.initialize().await {
            eprintln!("Warning: Failed to initialize event sender: {}", e);
        }
        // Spawn a new child process asynchronously
        // `mut` makes the variable mutable (can be modified)
        let mut child = AsyncCommand::new(&command.command) // `&` borrows command.command
            .args(&command.args) // Builder pattern - chain method calls
            .stdin(Stdio::piped()) // Pipe stdin (we can write to it)
            .stdout(Stdio::piped()) // Pipe stdout (we can read from it)
            .stderr(Stdio::inherit()) // Inherit stderr from parent
            .spawn()?; // `?` operator - return early if error

        // `take()` moves the value out of the Option, leaving None in its place
        // `unwrap()` extracts value from Option, panics if None
        let child_stdin = child.stdin.take().unwrap();
        let child_stdout = child.stdout.take().unwrap();

        // Get a mutable reference to stdout for writing output
        let mut stdout = io::stdout();

        // Shadow previous variable with same name (common Rust pattern)
        let mut child_stdin = child_stdin;
        // Create async buffered reader that yields lines
        let mut child_stdout_reader = AsyncBufReader::new(child_stdout).lines();

        // Clone needed because we're moving it into the async block below
        let log_repo_clone = LogRepository::new();
        let event_sender_clone = EventSender::new();
        if let Err(e) = event_sender_clone.initialize().await {
            eprintln!("Warning: Failed to initialize event sender clone: {}", e);
        }

        // `tokio::spawn` creates a new async task (like a lightweight thread)
        // `async move` creates an async block that takes ownership of captured variables
        let stdin_handle = tokio::spawn(async move {
            let stdin = io::stdin();
            let reader = BufReader::new(stdin);
            // Iterate over lines from stdin
            for line in reader.lines() {
                let line = line.unwrap();

                // Create a log entry for the request
                let entry = LogEntry::request(line.clone()); // `clone()` creates a copy of the String
                                                             // `let _` ignores the result (we don't care if logging fails)
                let _ = log_repo_clone.write_entry(&entry);

                // Create and send MCP event to API
                let method = extract_method_from_content(&line);
                let event = create_request_event(line.clone(), method);
                event_sender_clone.buffer_event(event).await;

                // Write line to child process stdin
                // `.await` pauses execution until the async operation completes
                // `as_bytes()` converts String to byte slice (&[u8])
                // `is_err()` checks if Result is an Err variant
                if child_stdin.write_all(line.as_bytes()).await.is_err() {
                    break; // Exit loop if write fails
                }
                // `b"\n"` is a byte string literal
                if child_stdin.write_all(b"\n").await.is_err() {
                    break;
                }
            }

            // Flush any remaining buffered events
            event_sender_clone.flush_buffer().await;
        });

        // Read lines from child process stdout
        // `while let` pattern - continues while pattern matches
        // `Ok(Some(line))` destructures nested Result and Option
        while let Ok(Some(line)) = child_stdout_reader.next_line().await {
            let entry = LogEntry::response(line.clone());
            let _ = log_repo.write_entry(&entry);

            // Create and send MCP response event to API
            let method = extract_method_from_content(&line);
            let event = create_response_event(line.clone(), method);
            event_sender.buffer_event(event).await;

            // `println!` macro - prints to stdout with newline
            // `{}` is a format placeholder
            println!("{}", line);
            // Flush stdout to ensure output is immediately visible
            let _ = stdout.flush();
        }

        // Abort the stdin reading task
        stdin_handle.abort();

        // Flush any remaining response events
        event_sender.flush_buffer().await;

        // Wait for child process to exit
        let _ = child.wait().await;

        // Return Ok with unit value () (empty tuple)
        Ok(())
    }
}
