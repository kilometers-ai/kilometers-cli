# Contributing to Kilometers CLI

Thank you for your interest in contributing to the Kilometers CLI project!

## Development Setup

### Prerequisites

- Go 1.24.5 or later
- Git

### Getting Started

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/your-username/kilometers-cli.git
   cd kilometers-cli
   ```

2. Set up development environment:
   ```bash
   # Install dependencies
   go mod download
   
   # Set up Git hooks (recommended)
   ./scripts/setup-hooks.sh
   ```

3. Build and test:
   ```bash
   # Build the CLI
   go build -o km ./cmd
   
   # Run tests
   go test ./...
   
   # Run static analysis
   go vet ./...
   ```

## Git Hooks

This project includes pre-commit hooks that automatically:

- Format Go code using `go fmt`
- Run static analysis with `go vet`  
- Tidy Go modules with `go mod tidy`

### Setting up hooks

Run the setup script to enable hooks:
```bash
./scripts/setup-hooks.sh
```

### Manual hook testing

Test the pre-commit hook manually:
```bash
./.githooks/pre-commit
```

### Bypassing hooks

If you need to commit without running hooks (not recommended):
```bash
git commit --no-verify
```

## Code Standards

- All Go code must be formatted with `go fmt`
- Code must pass `go vet` static analysis
- Dependencies should be managed with `go mod`
- Follow Go best practices and idioms

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes following the code standards
3. Test your changes thoroughly
4. Commit with clear, descriptive messages
5. Push to your fork and create a pull request

## IDE Setup

See [docs/IDE_SETUP.md](docs/IDE_SETUP.md) for GoLand/IntelliJ IDEA configuration.

## Plugin Development

This CLI supports a plugin architecture. See the `internal/plugins/` directory for examples and interfaces.

## Questions?

Feel free to open an issue for questions or discussions about contributing.