# Contributing to Kilometers CLI

Thank you for your interest in contributing to Kilometers CLI! We welcome contributions from developers of all skill levels and backgrounds.

## 🚀 Quick Start

1. **Fork** the repository on GitHub
2. **Clone** your fork locally
3. **Create** a new branch for your feature
4. **Make** your changes
5. **Test** thoroughly
6. **Submit** a pull request

## 🎯 Ways to Contribute

### 🐛 Report Bugs

Found a bug? Please create an issue with:

- **Clear title** describing the problem
- **Steps to reproduce** the issue
- **Expected vs actual behavior**
- **Environment details** (OS, Rust version, etc.)
- **Log output** (if relevant)

**Use the bug report template**: [Create Bug Report](https://github.com/kilometers-ai/kilometers-cli/issues/new?template=bug_report.md)

### ✨ Request Features

Have an idea for improvement? We'd love to hear it:

- **Use the feature request template**: [Request Feature](https://github.com/kilometers-ai/kilometers-cli/issues/new?template=feature_request.md)
- **Describe the problem** your feature would solve
- **Explain your proposed solution**
- **Consider alternative approaches**
- **Provide use cases** and examples

### 🔧 Code Contributions

#### Setting Up Development Environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/kilometers-cli.git
cd kilometers-cli

# Install Rust (if not already installed)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Install development tools
cargo install cargo-watch cargo-tarpaulin cargo-audit

# Set up environment
cp .env.example .env
# Edit .env with your development API key

# Verify setup
cargo build
cargo test
```

#### Making Changes

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   # or
   git checkout -b fix/bug-description
   ```

2. **Make your changes** following our coding standards

3. **Write tests** for new functionality

4. **Run the test suite**:
   ```bash
   cargo test
   cargo clippy
   cargo fmt -- --check
   ```

5. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add amazing new feature"
   ```

## 📋 Development Guidelines

### Code Style

We follow standard Rust conventions:

- **Use `cargo fmt`** for consistent formatting
- **Fix all `cargo clippy` warnings**
- **Follow [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/)**
- **Write clear, self-documenting code**
- **Add comments for complex logic**

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation only changes
- `style:` Changes that don't affect meaning (formatting, etc.)
- `refactor:` Code change that neither fixes a bug nor adds a feature
- `perf:` Performance improvement
- `test:` Adding missing tests or correcting existing tests
- `chore:` Changes to build process or auxiliary tools

**Examples:**
```bash
feat(auth): add JWT token refresh mechanism
fix(proxy): handle connection timeouts gracefully
docs(readme): update installation instructions
test(filters): add unit tests for risk analysis
chore(deps): update reqwest to 0.12
```

### Testing Requirements

All contributions must include appropriate tests:

- **Unit tests** for individual functions and methods
- **Integration tests** for complete workflows
- **Error case testing** for failure scenarios
- **Performance tests** for critical paths (when applicable)

```bash
# Run all tests
cargo test

# Run tests with coverage
cargo tarpaulin --out html --output-dir coverage/

# Run specific test files
cargo test test_auth
cargo test --test integration_tests
```

### Documentation

- **Document all public APIs** with rustdoc comments
- **Update README** if adding user-facing features
- **Add examples** for new functionality
- **Update CHANGELOG** (maintainers will handle this)

### Performance Considerations

- **Benchmark performance-critical code**
- **Consider memory usage** for long-running operations
- **Minimize allocations** where possible
- **Use async/await** appropriately

## 🔍 Code Review Process

### What We Look For

- **Correctness**: Does the code work as intended?
- **Test Coverage**: Are changes adequately tested?
- **Performance**: Any performance implications?
- **Security**: Are there security considerations?
- **Documentation**: Is the code well-documented?
- **Style**: Does it follow our conventions?

### Review Timeline

- **Initial response**: Within 2 business days
- **Full review**: Within 5 business days
- **Maintainer review**: For significant changes

### Addressing Feedback

- **Be responsive** to review comments
- **Ask questions** if feedback is unclear
- **Make requested changes** promptly
- **Update tests** if implementation changes

## 🛡️ Security

### Reporting Security Issues

**Do NOT create public issues for security vulnerabilities.**

Instead, email us at: security@kilometers.ai

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Security Best Practices

- **Never commit secrets** (API keys, passwords, etc.)
- **Validate all inputs** from external sources
- **Use secure defaults** for configurations
- **Follow least privilege principle**

## 🎉 Recognition

Contributors are recognized in several ways:

- **Contributors list** in README
- **Release notes** mention significant contributions
- **GitHub insights** track your contributions
- **Swag** for significant contributors (stickers, shirts)
- **Maintainer status** for long-term contributors

## 📚 Resources

### Documentation
- [Rust Book](https://doc.rust-lang.org/book/)
- [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/)
- [Tokio Tutorial](https://tokio.rs/tokio/tutorial)
- [MCP Specification](https://modelcontextprotocol.io/docs/specification)

### Project Resources
- [Architecture Overview](ARCHITECTURE.md) (if exists)
- [API Documentation](https://docs.rs/km)
- [Issue Templates](.github/ISSUE_TEMPLATE/)
- [CI/CD Pipeline](.github/workflows/)

### Getting Help

- **GitHub Discussions**: [Ask questions](https://github.com/kilometers-ai/kilometers-cli/discussions)
- **Discord**: [Join our community](https://discord.gg/kilometers)
- **Email**: contributors@kilometers.ai

## 🏆 Contributor Levels

### 🌱 First-time Contributors

New to open source? We've got you covered:

- Look for `good first issue` labels
- Start with documentation improvements
- Ask questions in Discord or discussions
- We provide extra guidance and support

### 🚀 Regular Contributors

- Help review other contributions
- Participate in feature discussions
- Mentor new contributors
- May be invited to join the contributors team

### 🎖️ Core Contributors

- Regular, significant contributions
- Deep understanding of the codebase
- Help with project direction and roadmap
- May be invited to become maintainers

## 📊 Development Workflow

### Issue Lifecycle

1. **Triage**: New issues are labeled and prioritized
2. **Discussion**: Requirements and approach clarified
3. **Implementation**: Code written and tested
4. **Review**: Pull request reviewed by maintainers
5. **Merge**: Approved changes merged to main
6. **Release**: Changes included in next release

### Release Process

Releases follow semantic versioning:

- **Patch** (0.1.1): Bug fixes
- **Minor** (0.2.0): New features, backwards compatible
- **Major** (1.0.0): Breaking changes

## 🤝 Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold this code.

Key points:
- **Be respectful** and inclusive
- **Welcome newcomers** and help them learn
- **Focus on what's best** for the community
- **Show empathy** towards others

## 📜 License

By contributing to Kilometers CLI, you agree that your contributions will be licensed under the [MIT License](LICENSE).

---

## 🙏 Thank You!

Every contribution, no matter how small, makes Kilometers CLI better for everyone. We appreciate your time and effort in making this project great!

**Happy coding!** 🚀
