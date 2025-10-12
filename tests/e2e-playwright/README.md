# CLI E2E Testing Guide

This directory contains end-to-end (E2E) tests for the Kilometers CLI using Playwright.

## Quick Start

### Prerequisites

- Node.js 20.x or higher
- Rust and Cargo (for building the CLI)
- npm 10.x or higher

### Installation

```bash
# Install Node dependencies
npm install

# Install Playwright browsers
npx playwright install chromium

# Build the CLI
cargo build --release
# or
cargo build  # for debug build
```

### Running Tests

```bash
# Run all E2E tests (builds CLI first)
npm run test:e2e

# Run tests in headed mode
npm run test:e2e:headed

# Debug tests
npm run test:e2e:debug

# Run specific test file
npx playwright test tests/e2e-playwright/init.spec.ts

# Run tests matching a pattern
npx playwright test --grep "init"
```

### View Test Reports

```bash
# Open HTML report
npm run test:e2e:report

# Or manually
npx playwright show-report playwright-report-cli
```

## Test Structure

```
tests/e2e-playwright/
├── init.spec.ts              # Init command tests
├── monitor.spec.ts           # Monitor command tests
├── clear-logs.spec.ts        # Clear-logs and misc tests
├── cross-integration.spec.ts # CLI ↔ Dashboard integration
├── helpers/
│   └── cli-runner.ts         # CLI testing utilities
└── README.md                 # This file
```

## Writing CLI Tests

### Basic Test Structure

```typescript
import { test, expect } from '@playwright/test';
import { CliRunner, createTempTestDir, cleanupTempTestDir } from './helpers/cli-runner';

test.describe('CLI Feature', () => {
  test('should do something', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['command', '--flag', 'value']);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('expected output');
  });
});
```

### Using Temporary Directories

```typescript
test('should create config file', async () => {
  const testDir = createTempTestDir();
  const cli = new CliRunner();

  try {
    const result = await cli.run(['init', '--api-key', 'test'], {
      env: { HOME: testDir },
    });

    expect(result.exitCode).toBe(0);
    // Verify files created in testDir
  } finally {
    cleanupTempTestDir(testDir);
  }
});
```

### Interactive CLI Tests

```typescript
test('should handle interactive input', async () => {
  const cli = new CliRunner();

  await cli.runInteractive(['monitor', '--', 'some-server']);
  await cli.sendInput('some input');
  await cli.waitForOutput('expected response');

  const stdout = cli.getStdout();
  expect(stdout).toContain('expected text');

  cli.kill();
});
```

## CLI Test Utilities

### CliRunner

Main utility for running CLI commands:

```typescript
const cli = new CliRunner();

// Run command and wait for completion
const result = await cli.run(['command'], {
  timeout: 30000,              // Max time to wait
  env: { KEY: 'value' },       // Environment variables
  input: 'stdin input',        // Standard input
});

// Access results
console.log(result.stdout);    // Standard output
console.log(result.stderr);    // Standard error
console.log(result.exitCode);  // Exit code

// Interactive mode
await cli.runInteractive(['command']);
await cli.sendInput('text');
await cli.waitForOutput('expected');
cli.kill();
```

### Temporary Directories

```typescript
import {
  createTempTestDir,
  cleanupTempTestDir,
  verifyFileExists,
  readFile,
  readJsonFile,
} from './helpers/cli-runner';

// Create temp directory
const dir = createTempTestDir();

// Use for tests...

// Clean up
cleanupTempTestDir(dir);
```

## Cross-Integration Tests

Tests that verify CLI → Dashboard integration:

```typescript
// cross-integration.spec.ts

test('should send telemetry to dashboard', async () => {
  const cli = new CliRunner();
  const browser = await chromium.launch();
  const page = await browser.newPage();

  // 1. Setup dashboard
  await page.goto('http://localhost:3000');
  await page.evaluate(() => {
    localStorage.setItem('apiKey', 'test-key');
  });

  // 2. Run CLI command that generates events
  await cli.run(['monitor', '--', 'mock-server']);

  // 3. Verify events appear in dashboard
  await page.goto('http://localhost:3000/events');
  await expect(page.locator('table tbody tr')).toHaveCount(1);

  await browser.close();
});
```

## Best Practices

### 1. Always Clean Up

```typescript
test.afterEach(() => {
  // Clean up temp files
  // Kill any running processes
});
```

### 2. Use Isolated Test Directories

```typescript
// Good - each test has its own directory
const testDir = createTempTestDir();

// Bad - sharing directories between tests
const testDir = '/tmp/shared';
```

### 3. Set Reasonable Timeouts

```typescript
// CLI operations can take longer than web UI
const result = await cli.run(['build'], {
  timeout: 60000, // 1 minute
});
```

### 4. Handle Process Cleanup

```typescript
const cli = new CliRunner();
try {
  await cli.runInteractive(['monitor']);
  // ... test code
} finally {
  cli.kill(); // Always kill interactive processes
}
```

## Debugging

### View CLI Output

```typescript
const result = await cli.run(['command']);
console.log('STDOUT:', result.stdout);
console.log('STDERR:', result.stderr);
console.log('EXIT CODE:', result.exitCode);
```

### Increase Verbosity

```typescript
const result = await cli.run(['command', '-vvv'], {
  env: {
    RUST_LOG: 'debug',
    RUST_BACKTRACE: '1',
  },
});
```

### Run Single Test

```bash
npx playwright test tests/e2e-playwright/init.spec.ts --debug
```

## CI/CD

Tests run automatically on:
- Push to `main` or `develop`
- Pull requests
- Manual workflow dispatch

Tests run on:
- Ubuntu (Linux)
- macOS
- Windows

## Troubleshooting

### "CLI binary not found"

```bash
# Build the CLI first
cargo build --release
# or
cargo build
```

### Tests Timeout

Increase timeout in test or config:

```typescript
// In test
test('slow operation', async () => {
  test.setTimeout(120000); // 2 minutes
});

// In playwright.config.ts
timeout: 60 * 1000, // 60 seconds
```

### Permission Errors

On Unix systems, ensure binary is executable:

```bash
chmod +x target/debug/km
chmod +x target/release/km
```

### Environment Variable Issues

Debug env vars:

```typescript
const result = await cli.run(['command'], {
  env: {
    ...process.env,  // Include existing env
    CUSTOM_VAR: 'value',
  },
});
```

## Testing the Mock MCP Server

The CLI includes a mock MCP server for testing:

```bash
# Build it
cargo build --bin mock_mcp_server

# Run it manually
./target/debug/mock_mcp_server

# Use in tests
const mockServerPath = path.join(__dirname, '../../target/debug/mock_mcp_server');
await cli.run(['monitor', '--', mockServerPath]);
```

## Additional Resources

- [Playwright Documentation](https://playwright.dev)
- [CLI Testing Best Practices](https://playwright.dev/docs/test-cli)
- [Rust Testing Guide](https://doc.rust-lang.org/book/ch11-00-testing.html)
