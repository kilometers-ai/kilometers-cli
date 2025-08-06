# 📜 **Script Reference Guide**

Complete reference for all development and deployment scripts in the Kilometers CLI project.

## 📂 **Script Organization**

```
scripts/
├── 🏗️ build/                    # Build and release automation
├── 🧪 test/                     # Testing and validation scripts
├── 📦 install/                  # Installation and setup scripts
└── 🔌 plugin/                   # Plugin development tools
```

---

## 🏗️ **Build Scripts** (`scripts/build/`)

### **`build-releases.sh`**

Automated multi-platform build script for creating release binaries.

**Usage:**
```bash
./scripts/build/build-releases.sh [OPTIONS]

Options:
  --platform PLATFORM    Build for specific platform (linux, darwin, windows)
  --version VERSION       Override version number
  --output DIR           Output directory (default: ./build)
  --help                 Show help message
```

**Examples:**
```bash
# Build all platforms
./scripts/build/build-releases.sh

# Build Linux only
./scripts/build/build-releases.sh --platform linux

# Custom version and output
./scripts/build/build-releases.sh --version v1.2.3 --output ./dist
```

**Output:**
- `build/km-linux-amd64`
- `build/km-darwin-amd64` 
- `build/km-darwin-arm64`
- `build/km-windows-amd64.exe`

---

## 🧪 **Test Scripts** (`scripts/test/`)

### **`run-tests.sh`**

Comprehensive test suite covering unit tests, integration tests, and code quality checks.

**Usage:**
```bash
./scripts/test/run-tests.sh [OPTIONS]

Options:
  --unit                 Run unit tests only
  --integration          Run integration tests only
  --coverage             Generate coverage report
  --verbose              Verbose output
  --help                 Show help message
```

**Test Coverage:**
- ✅ Unit tests for all packages
- ✅ Integration tests for CLI commands
- ✅ Plugin loading and communication tests
- ✅ MCP server proxy functionality
- ✅ Code quality and linting

**Examples:**
```bash
# Full test suite
./scripts/test/run-tests.sh

# Unit tests with coverage
./scripts/test/run-tests.sh --unit --coverage

# Verbose integration tests
./scripts/test/run-tests.sh --integration --verbose
```

### **`test-mcp-monitoring.sh`**

End-to-end testing of MCP server monitoring functionality.

**Usage:**
```bash
./scripts/test/test-mcp-monitoring.sh [OPTIONS]

Options:
  --server SERVER        MCP server command to test
  --timeout SECONDS      Test timeout (default: 30)
  --api-key KEY          API key for authenticated testing
  --debug                Enable debug logging
  --help                 Show help message
```

**Test Scenarios:**
- 🔍 Basic MCP message forwarding
- 📊 Console logging functionality  
- 🔑 API key authentication
- 🔌 Plugin loading and initialization
- ⚡ Performance and latency validation

**Examples:**
```bash
# Test with filesystem MCP server
./scripts/test/test-mcp-monitoring.sh --server "npx -y @modelcontextprotocol/server-filesystem /tmp"

# Test with API authentication
./scripts/test/test-mcp-monitoring.sh --api-key "km_live_test123" --debug
```

### **`test-plugin-integration.sh`**

Dedicated plugin system integration testing.

**Usage:**
```bash
./scripts/test/test-plugin-integration.sh [OPTIONS]

Options:
  --plugin PLUGIN        Test specific plugin (console-logger, api-logger)
  --no-auth              Skip authentication tests
  --build-plugins        Build plugins before testing
  --debug                Enable debug logging
  --help                 Show help message
```

**Plugin Tests:**
- 🔌 Plugin discovery and loading
- 🔑 Authentication and JWT validation
- 📨 Message handling and forwarding
- 🚫 Error handling and graceful failures
- 🔒 Security and permission validation

**Examples:**
```bash
# Test all plugins
./scripts/test/test-plugin-integration.sh --build-plugins

# Test specific plugin
./scripts/test/test-plugin-integration.sh --plugin console-logger --debug

# Skip authentication (local testing)
./scripts/test/test-plugin-integration.sh --no-auth
```

### **`test-and-check.sh`**

Quick development validation script for rapid feedback.

**Usage:**
```bash
./scripts/test/test-and-check.sh [OPTIONS]

Options:
  --fast                 Skip slow tests
  --fix                  Auto-fix formatting issues
  --help                 Show help message
```

**Quick Checks:**
- ⚡ Go compilation
- 🎨 Code formatting (`gofmt`)
- 🔍 Basic linting (`go vet`)
- 📝 Critical unit tests
- 🏗️ Build verification

**Examples:**
```bash
# Quick validation
./scripts/test/test-and-check.sh

# Fast mode (skip integration tests)
./scripts/test/test-and-check.sh --fast

# Auto-fix formatting
./scripts/test/test-and-check.sh --fix
```

---

## 📦 **Install Scripts** (`scripts/install/`)

### **`install.sh`** (Linux/macOS)

Cross-platform installation script for Unix-like systems.

**Usage:**
```bash
# Direct install
curl -fsSL https://install.kilometers.ai | sh

# Manual install
./scripts/install/install.sh [OPTIONS]

Options:
  --version VERSION      Install specific version
  --prefix PREFIX        Installation prefix (default: /usr/local)
  --binary-name NAME     Binary name (default: km)
  --help                 Show help message
```

**Installation Process:**
1. 🔍 Detect platform and architecture
2. 📥 Download appropriate binary
3. ✅ Verify binary signature (if available)
4. 📂 Install to system PATH
5. 🔧 Create shell completions
6. ✅ Verify installation

**Examples:**
```bash
# Standard installation
./scripts/install/install.sh

# Install specific version
./scripts/install/install.sh --version v1.2.3

# Custom installation directory
./scripts/install/install.sh --prefix $HOME/.local
```

### **`install.ps1`** (Windows)

PowerShell installation script for Windows systems.

**Usage:**
```powershell
# Direct install
iwr -useb https://install.kilometers.ai/windows | iex

# Manual install
.\scripts\install\install.ps1 [OPTIONS]

Parameters:
  -Version VERSION       Install specific version
  -InstallDir DIR        Installation directory
  -AddToPath            Add to system PATH
  -Help                 Show help message
```

**Installation Features:**
- 🪟 Windows-specific binary detection
- 🛡️ PowerShell execution policy handling
- 📂 Program Files installation
- 🔧 Registry PATH modification
- ✅ Installation verification

**Examples:**
```powershell
# Standard installation
.\scripts\install\install.ps1

# Install to custom directory
.\scripts\install\install.ps1 -InstallDir "C:\Tools" -AddToPath

# Install specific version
.\scripts\install\install.ps1 -Version "v1.2.3"
```

---

## 🔌 **Plugin Scripts** (`scripts/plugin/`)

### **`build-plugin.sh`**

Secure plugin binary builder with customer-specific signing.

**Usage:**
```bash
./scripts/plugin/build-plugin.sh [OPTIONS]

Required:
  --plugin PLUGIN        Plugin name (console-logger, api-logger)
  --customer CUSTOMER    Customer identifier
  --api-key API_KEY      Customer API key

Options:
  --tier TIER           Subscription tier (Free, Pro, Enterprise)
  --output DIR          Output directory (default: ./dist)
  --sign                Enable digital signing
  --debug               Enable debug output
  --help                Show help message
```

**Build Process:**
1. 🔍 Validate plugin source code
2. 🔑 Embed customer-specific credentials
3. 🏗️ Compile plugin binary
4. 📝 Generate digital signature
5. 📦 Package plugin with manifest
6. ✅ Verify package integrity

**Examples:**
```bash
# Build console logger for customer
./scripts/plugin/build-plugin.sh \
  --plugin console-logger \
  --customer demo_customer_123 \
  --api-key km_live_demo123456789

# Build Pro API logger with signing
./scripts/plugin/build-plugin.sh \
  --plugin api-logger \
  --customer pro_customer_456 \
  --api-key km_live_pro987654321 \
  --tier Pro \
  --sign \
  --debug
```

**Output Files:**
- `dist/km-plugin-{plugin}-{hash}.kmpkg` - Signed plugin package
- `dist/km-plugin-{plugin}-{hash}.sig` - Digital signature
- `dist/km-plugin-{plugin}-{hash}.json` - Manifest file

### **`verify-plugin.sh`**

Plugin package verification and integrity checking.

**Usage:**
```bash
./scripts/plugin/verify-plugin.sh PLUGIN_PACKAGE [OPTIONS]

Options:
  --public-key KEY      Public key for signature verification
  --verbose             Verbose verification output
  --extract DIR         Extract plugin to directory
  --help                Show help message
```

**Verification Process:**
1. 📦 Extract plugin package
2. 🔍 Validate package structure
3. 📝 Verify digital signature
4. 🔑 Check embedded credentials
5. ✅ Validate manifest integrity

**Examples:**
```bash
# Basic verification
./scripts/plugin/verify-plugin.sh dist/km-plugin-console-logger-abc123.kmpkg

# Verbose verification with extraction
./scripts/plugin/verify-plugin.sh \
  dist/km-plugin-api-logger-def456.kmpkg \
  --verbose \
  --extract /tmp/plugin-check
```

### **`install-plugin.sh`**

Plugin installation and system integration.

**Usage:**
```bash
./scripts/plugin/install-plugin.sh PLUGIN_PACKAGE [OPTIONS]

Options:
  --install-dir DIR     Plugin installation directory
  --system              Install system-wide
  --user                Install for current user only
  --force               Force overwrite existing plugin
  --help                Show help message
```

**Installation Process:**
1. ✅ Verify plugin package integrity
2. 🔍 Check system compatibility
3. 📂 Extract to installation directory
4. 🔧 Register plugin with system
5. ✅ Test plugin functionality

**Examples:**
```bash
# Install plugin for current user
./scripts/plugin/install-plugin.sh km-plugin-console-logger-abc123.kmpkg --user

# System-wide installation
./scripts/plugin/install-plugin.sh km-plugin-api-logger-def456.kmpkg --system

# Force reinstall
./scripts/plugin/install-plugin.sh plugin.kmpkg --force
```

### **`demo-security-model.sh`**

Demonstration of the plugin security architecture and workflow.

**Usage:**
```bash
./scripts/plugin/demo-security-model.sh [OPTIONS]

Options:
  --interactive         Interactive demonstration mode
  --step-by-step        Execute with pauses between steps
  --no-cleanup          Don't clean up demo files
  --help                Show help message
```

**Demonstration Features:**
- 🔐 Customer-specific binary generation
- 📝 Digital signature creation and verification
- 🎫 JWT token embedding and validation
- 🔒 Tamper detection demonstration
- 🚫 Unauthorized access prevention

**Demo Scenarios:**
1. **Valid Plugin**: Successful authentication and operation
2. **Tampered Binary**: Detection of unauthorized modifications
3. **Wrong Customer**: Plugin rejection for different customer
4. **Expired Token**: Handling of expired authentication
5. **Invalid Signature**: Detection of signature tampering

**Examples:**
```bash
# Full security demonstration
./scripts/plugin/demo-security-model.sh

# Interactive step-by-step demo
./scripts/plugin/demo-security-model.sh --interactive --step-by-step

# Demo with file preservation
./scripts/plugin/demo-security-model.sh --no-cleanup
```

---

## 🔄 **Script Dependencies**

### **System Requirements**

- **Go 1.21+**: Required for all build scripts
- **OpenSSL**: Required for plugin signing (`plugin/` scripts)
- **jq**: Required for JSON processing in plugin scripts
- **curl/wget**: Required for installation scripts
- **tar/zip**: Required for packaging operations

### **Environment Variables**

```bash
# Development
export KM_DEBUG=true              # Enable debug logging
export KM_PLUGIN_DEBUG=true       # Enable plugin debug logging
export KM_API_ENDPOINT=...        # Override API endpoint

# Build
export VERSION=v1.2.3             # Override version for builds
export BUILD_TAGS="..."           # Additional build tags

# Testing
export KM_TEST_API_KEY=...        # API key for testing
export KM_TEST_TIMEOUT=60         # Test timeout in seconds

# Plugin Development
export KM_PLUGIN_DIR=...          # Plugin directory override
export KM_SIGNING_KEY=...         # Plugin signing key path
```

### **Script Execution Order**

For complete development workflow:

1. **Setup**: `./scripts/install/install.sh` (if needed)
2. **Development**: `./scripts/test/test-and-check.sh` (rapid feedback)
3. **Testing**: `./scripts/test/run-tests.sh` (comprehensive testing)
4. **Plugin Testing**: `./scripts/test/test-plugin-integration.sh`
5. **Build**: `./scripts/build/build-releases.sh`
6. **Plugin Build**: `./scripts/plugin/build-plugin.sh` (if developing plugins)

---

## 🐛 **Troubleshooting Scripts**

### **Common Script Issues**

1. **Permission Denied**
   ```bash
   chmod +x scripts/**/*.sh
   ```

2. **Missing Dependencies**
   ```bash
   # Install required tools
   sudo apt-get install jq openssl curl tar  # Ubuntu/Debian
   brew install jq openssl curl              # macOS
   ```

3. **Path Issues**
   ```bash
   # Run from project root
   cd kilometers-cli
   ./scripts/test/run-tests.sh
   ```

4. **Environment Variables**
   ```bash
   # Check required variables
   echo $KM_API_KEY
   echo $KM_API_ENDPOINT
   ```

### **Script Debugging**

```bash
# Enable bash debugging
bash -x ./scripts/test/run-tests.sh

# Check script syntax
bash -n ./scripts/test/run-tests.sh

# Trace script execution
set -x
./scripts/test/run-tests.sh
set +x
```

---

## 📚 **Additional Resources**

- **[Getting Started Guide](GETTING_STARTED.md)** - Development setup
- **[Architecture Guide](ARCHITECTURE.md)** - System design
- **[Plugin Development](../plugins/DEVELOPMENT.md)** - Plugin creation
- **[Contributing Guide](CONTRIBUTING.md)** - Contribution workflow

---

**For questions about scripts, check the script source code or create an issue on GitHub.**