#!/usr/bin/env python3
"""
Mock GitHub Releases API server for testing install scripts.
Simulates various scenarios including failures and edge cases.
"""

import json
import os
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import argparse
import threading
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MockGitHubHandler(BaseHTTPRequestHandler):
    """Handler for mock GitHub API requests."""

    def __init__(self, *args, test_mode="normal", **kwargs):
        self.test_mode = test_mode
        self.responses = self.load_responses()
        super().__init__(*args, **kwargs)

    def load_responses(self):
        """Load mock response data."""
        return {
            "releases/latest": {
                "tag_name": "v2024.1.1",
                "name": "Release v2024.1.1",
                "assets": [
                    {
                        "name": "km-x86_64-unknown-linux-gnu.tar.gz",
                        "browser_download_url": f"http://{os.getenv('MOCK_SERVER_HOST', 'localhost')}:8080/releases/download/v2024.1.1/km-x86_64-unknown-linux-gnu.tar.gz"
                    },
                    {
                        "name": "km-aarch64-unknown-linux-gnu.tar.gz",
                        "browser_download_url": f"http://{os.getenv('MOCK_SERVER_HOST', 'localhost')}:8080/releases/download/v2024.1.1/km-aarch64-unknown-linux-gnu.tar.gz"
                    },
                    {
                        "name": "km-x86_64-apple-darwin.tar.gz",
                        "browser_download_url": f"http://{os.getenv('MOCK_SERVER_HOST', 'localhost')}:8080/releases/download/v2024.1.1/km-x86_64-apple-darwin.tar.gz"
                    },
                    {
                        "name": "km-aarch64-apple-darwin.tar.gz",
                        "browser_download_url": f"http://{os.getenv('MOCK_SERVER_HOST', 'localhost')}:8080/releases/download/v2024.1.1/km-aarch64-apple-darwin.tar.gz"
                    },
                    # Legacy naming for compatibility with existing install scripts
                    {
                        "name": "km-linux-amd64.tar.gz",
                        "browser_download_url": f"http://{os.getenv('MOCK_SERVER_HOST', 'localhost')}:8080/releases/download/v2024.1.1/km-linux-amd64.tar.gz"
                    },
                    {
                        "name": "km-linux-arm64.tar.gz",
                        "browser_download_url": f"http://{os.getenv('MOCK_SERVER_HOST', 'localhost')}:8080/releases/download/v2024.1.1/km-linux-arm64.tar.gz"
                    },
                    {
                        "name": "km-darwin-amd64.tar.gz",
                        "browser_download_url": f"http://{os.getenv('MOCK_SERVER_HOST', 'localhost')}:8080/releases/download/v2024.1.1/km-darwin-amd64.tar.gz"
                    },
                    {
                        "name": "km-darwin-arm64.tar.gz",
                        "browser_download_url": f"http://{os.getenv('MOCK_SERVER_HOST', 'localhost')}:8080/releases/download/v2024.1.1/km-darwin-arm64.tar.gz"
                    }
                ]
            }
        }

    def do_GET(self):
        """Handle GET requests."""
        parsed_url = urlparse(self.path)
        path = parsed_url.path.lstrip('/')

        logger.info(f"Request: {self.path} (mode: {self.test_mode})")

        # Handle different test modes
        if self.test_mode == "timeout":
            time.sleep(30)  # Simulate timeout
            return
        elif self.test_mode == "rate_limit":
            self.send_response(403)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({
                "message": "API rate limit exceeded"
            }).encode())
            return
        elif self.test_mode == "server_error":
            self.send_response(500)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({
                "message": "Internal server error"
            }).encode())
            return

        # Handle API endpoints
        logger.info(f"Routing path: {path}")
        if path.startswith('repos/kilometers-ai/kilometers-cli/releases/latest'):
            logger.info("Routing to latest release handler")
            self.handle_latest_release()
        elif path.startswith('releases/download/'):
            logger.info("Routing to binary download handler")
            self.handle_binary_download(path)
        else:
            logger.info(f"No handler found for path: {path}")
            self.send_response(404)
            self.end_headers()

    def handle_latest_release(self):
        """Handle latest release API request."""
        if self.test_mode == "malformed_json":
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(b'{"invalid": json}')  # Malformed JSON
            return

        response = self.responses["releases/latest"]
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(response).encode())

    def handle_binary_download(self, path):
        """Handle binary download requests."""
        # Extract filename from path
        filename = path.split('/')[-1]
        logger.info(f"Handling binary download: path={path}, filename={filename}")

        if self.test_mode == "corrupted_binary":
            # Send corrupted data
            self.send_response(200)
            self.send_header('Content-type', 'application/gzip')
            self.send_header('Content-Length', '100')
            self.end_headers()
            self.wfile.write(b'corrupted data' * 10)  # Not a valid tar.gz
            return
        elif self.test_mode == "missing_binary":
            self.send_response(404)
            self.end_headers()
            return

        # Check if we have a test binary for this filename
        data_dir = os.getenv('DATA_DIR', '/app/data')
        binary_path = os.path.join(data_dir, filename)
        logger.info(f"Looking for binary at: {binary_path}, exists: {os.path.exists(binary_path)}")
        if os.path.exists(binary_path):
            with open(binary_path, 'rb') as f:
                content = f.read()

            self.send_response(200)
            self.send_header('Content-type', 'application/gzip')
            self.send_header('Content-Length', str(len(content)))
            self.end_headers()
            self.wfile.write(content)
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format, *args):
        """Override to use logger."""
        logger.info(f"{self.client_address[0]} - {format % args}")

def create_handler(test_mode):
    """Create a handler class with the specified test mode."""
    class Handler(MockGitHubHandler):
        def __init__(self, *args, **kwargs):
            super().__init__(*args, test_mode=test_mode, **kwargs)
    return Handler

def start_server(port=8080, test_mode="normal"):
    """Start the mock server."""
    handler_class = create_handler(test_mode)
    server = HTTPServer(('0.0.0.0', port), handler_class)

    logger.info(f"Starting mock GitHub server on port {port} (mode: {test_mode})")
    logger.info("Available endpoints:")
    logger.info("  GET /repos/kilometers-ai/kilometers-cli/releases/latest")
    logger.info("  GET /releases/download/v2024.1.1/*.tar.gz")

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("Shutting down server...")
        server.shutdown()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Mock GitHub Releases API server")
    parser.add_argument("--port", type=int, default=8080, help="Port to listen on")
    parser.add_argument("--mode", default="normal",
                       choices=["normal", "timeout", "rate_limit", "server_error",
                               "malformed_json", "corrupted_binary", "missing_binary"],
                       help="Test mode to simulate different scenarios")

    args = parser.parse_args()
    start_server(args.port, args.mode)
