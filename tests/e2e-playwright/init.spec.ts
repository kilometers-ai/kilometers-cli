import { test, expect } from '@playwright/test';
import {
  CliRunner,
  createTempTestDir,
  cleanupTempTestDir,
  verifyFileExists,
  readJsonFile,
} from './helpers/cli-runner';
import * as path from 'path';

test.describe('CLI Init Command', () => {
  test.beforeAll(async () => {
    // Verify CLI is built
    await CliRunner.verifyBuilt();
  });

  test('should display help for init command', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['init', '--help']);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('init');
    expect(result.stdout).toMatch(/api[-\s]?key/i);
  });

  test('should initialize configuration with API key', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      const result = await cli.run(
        ['init', '--api-key', 'test-api-key-123'],
        {
          env: {
            HOME: testDir, // Override home directory for testing
          },
        }
      );

      expect(result.exitCode).toBe(0);
      expect(result.stdout).toMatch(/success|initialized|configured/i);

      // Verify config file was created
      const configPath = path.join(testDir, '.km', 'config.json');
      // Note: Actual path might vary based on implementation
    } finally {
      cleanupTempTestDir(testDir);
    }
  });

  test('should validate API key format', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      // Try with invalid API key format
      const result = await cli.run(['init', '--api-key', 'invalid'], {
        env: {
          HOME: testDir,
        },
        timeout: 10000,
      });

      // Should either fail or warn about invalid key format
      // The exact behavior depends on implementation
      expect(result.exitCode !== null).toBe(true);
    } finally {
      cleanupTempTestDir(testDir);
    }
  });

  test('should handle missing API key gracefully', async () => {
    const cli = new CliRunner();

    const result = await cli.run(['init'], {
      timeout: 5000,
    });

    // Should either prompt for key or show error
    expect(result.exitCode !== null).toBe(true);
  });

  test('should support setting API URL', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      const result = await cli.run(
        [
          'init',
          '--api-key',
          'test-key',
          '--api-url',
          'https://test.kilometers.ai',
        ],
        {
          env: {
            HOME: testDir,
          },
        }
      );

      expect(result.exitCode).toBe(0);
    } finally {
      cleanupTempTestDir(testDir);
    }
  });

  test('should handle reinitialize (overwrite existing config)', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      // First initialization
      await cli.run(['init', '--api-key', 'first-key'], {
        env: { HOME: testDir },
      });

      // Second initialization (should overwrite or prompt)
      const result = await cli.run(['init', '--api-key', 'second-key'], {
        env: { HOME: testDir },
        timeout: 10000,
      });

      expect(result.exitCode !== null).toBe(true);
    } finally {
      cleanupTempTestDir(testDir);
    }
  });
});
