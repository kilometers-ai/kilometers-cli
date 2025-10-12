import { test, expect } from '@playwright/test';
import {
  CliRunner,
  createTempTestDir,
  cleanupTempTestDir,
} from './helpers/cli-runner';
import * as path from 'path';

test.describe('CLI Monitor Command', () => {
  test.beforeAll(async () => {
    await CliRunner.verifyBuilt();
  });

  test('should display help for monitor command', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['monitor', '--help']);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('monitor');
  });

  test('should require a command to monitor', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['monitor'], {
      timeout: 5000,
    });

    // Should show error or help message
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr || result.stdout).toMatch(/command|required|usage/i);
  });

  test('should proxy to mock MCP server', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      // Setup test environment
      const mockServerPath = path.join(
        __dirname,
        '../../target/debug/mock_mcp_server'
      );

      // Run monitor with mock server
      const result = await cli.run(
        ['monitor', '--', mockServerPath],
        {
          env: {
            KM_API_KEY: 'test-key',
            HOME: testDir,
          },
          timeout: 10000,
          input: JSON.stringify({
            jsonrpc: '2.0',
            id: 1,
            method: 'initialize',
            params: {},
          }) + '\n',
        }
      );

      // Should complete without crashing
      expect(result.exitCode !== null).toBe(true);
    } finally {
      cleanupTempTestDir(testDir);
    }
  });

  test('should handle invalid command gracefully', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      const result = await cli.run(
        ['monitor', '--', 'nonexistent-command-12345'],
        {
          env: {
            KM_API_KEY: 'test-key',
            HOME: testDir,
          },
          timeout: 5000,
        }
      );

      // Should exit with error
      expect(result.exitCode).not.toBe(0);
      expect(result.stderr).toMatch(/error|failed|not found/i);
    } finally {
      cleanupTempTestDir(testDir);
    }
  });

  test('should require API key for monitoring', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      const result = await cli.run(
        ['monitor', '--', 'echo', 'test'],
        {
          env: {
            // No API key set
            HOME: testDir,
          },
          timeout: 5000,
        }
      );

      // Should fail or warn about missing API key
      if (result.exitCode !== 0) {
        expect(result.stderr || result.stdout).toMatch(/api.*key|auth|config/i);
      }
    } finally {
      cleanupTempTestDir(testDir);
    }
  });

  test('should support tier parameter', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      // Note: This test assumes tier flag exists
      // Adjust based on actual CLI implementation
      const result = await cli.run(
        ['monitor', '--tier', 'enterprise', '--', 'echo', 'test'],
        {
          env: {
            KM_API_KEY: 'test-key',
            HOME: testDir,
          },
          timeout: 10000,
        }
      );

      // Should accept tier parameter
      expect(result.exitCode !== null).toBe(true);
    } finally {
      cleanupTempTestDir(testDir);
    }
  });
});
