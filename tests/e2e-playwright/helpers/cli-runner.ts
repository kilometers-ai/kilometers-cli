import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import { expect } from '@playwright/test';

/**
 * Helper class for running and interacting with the km CLI
 */
export class CliRunner {
  private process: ChildProcess | null = null;
  private stdout: string = '';
  private stderr: string = '';
  private cliPath: string;

  constructor() {
    // Determine CLI binary path (debug or release)
    const debugPath = path.join(__dirname, '../../../target/debug/km');
    const releasePath = path.join(__dirname, '../../../target/release/km');

    if (fs.existsSync(releasePath)) {
      this.cliPath = releasePath;
    } else if (fs.existsSync(debugPath)) {
      this.cliPath = debugPath;
    } else {
      throw new Error(
        'CLI binary not found. Please build the project first with `cargo build`'
      );
    }
  }

  /**
   * Run a CLI command and wait for it to complete
   */
  async run(args: string[], options: {
    timeout?: number;
    env?: Record<string, string>;
    input?: string;
  } = {}): Promise<{ stdout: string; stderr: string; exitCode: number | null }> {
    const { timeout = 30000, env = {}, input } = options;

    return new Promise((resolve, reject) => {
      this.stdout = '';
      this.stderr = '';

      const processEnv = {
        ...process.env,
        ...env,
      };

      this.process = spawn(this.cliPath, args, {
        env: processEnv,
        stdio: input ? 'pipe' : 'inherit',
      });

      if (this.process.stdout) {
        this.process.stdout.on('data', (data) => {
          this.stdout += data.toString();
        });
      }

      if (this.process.stderr) {
        this.process.stderr.on('data', (data) => {
          this.stderr += data.toString();
        });
      }

      // Send input if provided
      if (input && this.process.stdin) {
        this.process.stdin.write(input);
        this.process.stdin.end();
      }

      const timeoutId = setTimeout(() => {
        this.kill();
        reject(new Error(`Command timed out after ${timeout}ms`));
      }, timeout);

      this.process.on('close', (code) => {
        clearTimeout(timeoutId);
        resolve({
          stdout: this.stdout,
          stderr: this.stderr,
          exitCode: code,
        });
      });

      this.process.on('error', (err) => {
        clearTimeout(timeoutId);
        reject(err);
      });
    });
  }

  /**
   * Run a CLI command in interactive mode (keeps process running)
   */
  async runInteractive(args: string[], env: Record<string, string> = {}): Promise<void> {
    this.stdout = '';
    this.stderr = '';

    const processEnv = {
      ...process.env,
      ...env,
    };

    this.process = spawn(this.cliPath, args, {
      env: processEnv,
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    if (this.process.stdout) {
      this.process.stdout.on('data', (data) => {
        this.stdout += data.toString();
      });
    }

    if (this.process.stderr) {
      this.process.stderr.on('data', (data) => {
        this.stderr += data.toString();
      });
    }

    // Wait a bit for process to start
    await this.waitForMs(500);
  }

  /**
   * Send input to interactive process
   */
  async sendInput(text: string): Promise<void> {
    if (!this.process || !this.process.stdin) {
      throw new Error('No interactive process running');
    }
    this.process.stdin.write(text + '\n');
    await this.waitForMs(100);
  }

  /**
   * Wait for specific output text
   */
  async waitForOutput(text: string, timeout: number = 10000): Promise<void> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
      if (this.stdout.includes(text) || this.stderr.includes(text)) {
        return;
      }
      await this.waitForMs(100);
    }

    throw new Error(
      `Timeout waiting for output: "${text}". Got: ${this.stdout} ${this.stderr}`
    );
  }

  /**
   * Get current stdout
   */
  getStdout(): string {
    return this.stdout;
  }

  /**
   * Get current stderr
   */
  getStderr(): string {
    return this.stderr;
  }

  /**
   * Kill the running process
   */
  kill(): void {
    if (this.process) {
      this.process.kill();
      this.process = null;
    }
  }

  /**
   * Check if CLI binary exists
   */
  static exists(): boolean {
    const debugPath = path.join(__dirname, '../../../target/debug/km');
    const releasePath = path.join(__dirname, '../../../target/release/km');

    return fs.existsSync(debugPath) || fs.existsSync(releasePath);
  }

  /**
   * Verify CLI is built
   */
  static async verifyBuilt(): Promise<void> {
    if (!CliRunner.exists()) {
      throw new Error(
        'CLI binary not found. Please build the project first:\n' +
        '  cargo build --release\n' +
        'or\n' +
        '  cargo build'
      );
    }
  }

  private async waitForMs(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}

/**
 * Helper to create temporary test directory
 */
export function createTempTestDir(): string {
  const tmpDir = path.join(__dirname, `../../../test-tmp-${Date.now()}`);
  fs.mkdirSync(tmpDir, { recursive: true });
  return tmpDir;
}

/**
 * Helper to clean up temporary test directory
 */
export function cleanupTempTestDir(dir: string): void {
  if (fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

/**
 * Helper to verify file exists
 */
export function verifyFileExists(filePath: string): void {
  expect(fs.existsSync(filePath)).toBe(true);
}

/**
 * Helper to read file contents
 */
export function readFile(filePath: string): string {
  return fs.readFileSync(filePath, 'utf-8');
}

/**
 * Helper to parse JSON file
 */
export function readJsonFile(filePath: string): any {
  const content = readFile(filePath);
  return JSON.parse(content);
}
