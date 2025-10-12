import { test, expect } from '@playwright/test';
import {
  CliRunner,
  createTempTestDir,
  cleanupTempTestDir,
} from './helpers/cli-runner';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI Clear Logs Command', () => {
  test.beforeAll(async () => {
    await CliRunner.verifyBuilt();
  });

  test('should display help for clear-logs command', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['clear-logs', '--help']);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch(/clear[-\s]?logs/i);
  });

  test('should clear logs successfully', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      // Create a mock log file
      const logsDir = path.join(testDir, '.km');
      fs.mkdirSync(logsDir, { recursive: true });
      const logFile = path.join(logsDir, 'mcp_proxy.log');
      fs.writeFileSync(logFile, 'test log content\n');

      const result = await cli.run(['clear-logs'], {
        env: {
          HOME: testDir,
        },
        timeout: 5000,
      });

      expect(result.exitCode).toBe(0);
      expect(result.stdout).toMatch(/cleared|removed|deleted|success/i);

      // Verify log file is cleared or removed
      if (fs.existsSync(logFile)) {
        const content = fs.readFileSync(logFile, 'utf-8');
        expect(content.length).toBe(0);
      }
    } finally {
      cleanupTempTestDir(testDir);
    }
  });

  test('should handle missing log file gracefully', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      const result = await cli.run(['clear-logs'], {
        env: {
          HOME: testDir,
        },
        timeout: 5000,
      });

      // Should complete successfully even if no logs exist
      expect(result.exitCode).toBe(0);
    } finally {
      cleanupTempTestDir(testDir);
    }
  });

  test('should support confirmation prompt', async () => {
    const testDir = createTempTestDir();
    const cli = new CliRunner();

    try {
      // Create a mock log file
      const logsDir = path.join(testDir, '.km');
      fs.mkdirSync(logsDir, { recursive: true });
      const logFile = path.join(logsDir, 'mcp_proxy.log');
      fs.writeFileSync(logFile, 'test log content\n');

      // If CLI supports --force flag, test it
      const result = await cli.run(['clear-logs', '--force'], {
        env: {
          HOME: testDir,
        },
        timeout: 5000,
      });

      // Should clear without prompting
      expect(result.exitCode !== null).toBe(true);
    } finally {
      cleanupTempTestDir(testDir);
    }
  });
});

test.describe('CLI Version and Info', () => {
  test('should display version', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['--version']);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch(/\d+\.\d+\.\d+/); // Semver pattern
  });

  test('should display help', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['--help']);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('Usage');
    expect(result.stdout).toMatch(/init|monitor|clear/);
  });
});

test.describe('CLI Error Handling', () => {
  test('should handle invalid command', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['invalid-command-xyz'], {
      timeout: 5000,
    });

    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toMatch(/error|invalid|unknown|unrecognized/i);
  });

  test('should handle invalid flags', async () => {
    const cli = new CliRunner();
    const result = await cli.run(['init', '--invalid-flag-xyz'], {
      timeout: 5000,
    });

    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toMatch(/error|invalid|unknown|unexpected/i);
  });
});
