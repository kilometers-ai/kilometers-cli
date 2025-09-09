# Kilometers CLI installer for Windows PowerShell
param(
    [string]$Version = "",
    [string]$InstallDir = ""
)

# Configuration
$BinaryName = "km.exe"
$GitHubRepo = "kilometers-ai/kilometers-cli"

# Default install directory
if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "bin"
}

# Function to write colored output
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
    exit 1
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { 
            Write-Error "Unsupported architecture: $arch"
        }
    }
}

# Get latest version from GitHub API
function Get-LatestVersion {
    Write-Info "Fetching latest release information..."
    
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$GitHubRepo/releases/latest"
        $latestVersion = $response.tag_name
        Write-Info "Latest version: $latestVersion"
        return $latestVersion
    }
    catch {
        Write-Error "Failed to fetch latest version: $($_.Exception.Message)"
    }
}

# Download and extract binary
function Install-Binary {
    param(
        [string]$Version,
        [string]$Architecture
    )
    
    $filename = "km-windows-$Architecture.zip"
    $url = "https://github.com/$GitHubRepo/releases/download/$Version/$filename"
    $tempDir = [System.IO.Path]::GetTempPath()
    $zipPath = Join-Path $tempDir $filename
    $extractPath = Join-Path $tempDir "km-extract"
    
    Write-Info "Downloading $filename..."
    
    try {
        # Download the zip file
        Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing
        
        # Create extraction directory
        if (Test-Path $extractPath) {
            Remove-Item -Path $extractPath -Recurse -Force
        }
        New-Item -ItemType Directory -Path $extractPath -Force | Out-Null
        
        Write-Info "Extracting binary..."
        
        # Extract zip file
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractPath)
        
        # Find the binary
        $binaryPath = Join-Path $extractPath $BinaryName
        if (-not (Test-Path $binaryPath)) {
            Write-Error "Binary not found in archive"
        }
        
        # Create install directory
        if (-not (Test-Path $InstallDir)) {
            Write-Info "Creating install directory: $InstallDir"
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        
        # Install binary
        $installPath = Join-Path $InstallDir $BinaryName
        Write-Info "Installing to $installPath..."
        Copy-Item -Path $binaryPath -Destination $installPath -Force
        
        # Cleanup
        Remove-Item -Path $zipPath -Force -ErrorAction SilentlyContinue
        Remove-Item -Path $extractPath -Recurse -Force -ErrorAction SilentlyContinue
        
        return $installPath
    }
    catch {
        Write-Error "Failed to download or install binary: $($_.Exception.Message)"
    }
}

# Check if directory is in PATH
function Test-InPath {
    param([string]$Directory)
    
    $pathDirs = $env:PATH -split ';'
    return $pathDirs -contains $Directory
}

# Add directory to PATH
function Add-ToPath {
    param([string]$Directory)
    
    if (Test-InPath $Directory) {
        Write-Info "$Directory is already in PATH"
        return
    }
    
    Write-Info "Adding $Directory to PATH..."
    
    try {
        # Get current user PATH
        $userPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
        
        if ($userPath) {
            $newPath = "$userPath;$Directory"
        } else {
            $newPath = $Directory
        }
        
        # Set new PATH
        [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
        
        # Update current session PATH
        $env:PATH = "$env:PATH;$Directory"
        
        Write-Info "Added $Directory to PATH"
        Write-Warning "You may need to restart your terminal for PATH changes to take effect"
    }
    catch {
        Write-Warning "Failed to add directory to PATH: $($_.Exception.Message)"
        Write-Warning "Please manually add $Directory to your PATH"
    }
}

# Verify installation
function Test-Installation {
    param([string]$BinaryPath)
    
    if (-not (Test-Path $BinaryPath)) {
        Write-Error "Installation failed: binary not found at $BinaryPath"
    }
    
    Write-Info "Installation successful!"
    Write-Info "Binary location: $BinaryPath"
    
    # Test if binary works
    try {
        $output = & $BinaryPath --help 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Info "Binary is working correctly"
        }
    }
    catch {
        Write-Warning "Binary may not be working correctly"
    }
    
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Green
    Write-Host "1. Run: " -NoNewline; Write-Host "km init" -ForegroundColor Yellow -NoNewline; Write-Host " to configure your API key"
    Write-Host "2. Start monitoring: " -NoNewline; Write-Host "km monitor -- <your-mcp-command>" -ForegroundColor Yellow
}

# Main execution
function Main {
    Write-Info "Installing Kilometers CLI for Windows..."
    
    # Detect architecture
    $arch = Get-Architecture
    Write-Info "Detected architecture: $arch"
    
    # Get version
    if (-not $Version) {
        $Version = Get-LatestVersion
    } else {
        Write-Info "Using specified version: $Version"
    }
    
    # Install binary
    $binaryPath = Install-Binary -Version $Version -Architecture $arch
    
    # Add to PATH
    Add-ToPath -Directory $InstallDir
    
    # Verify installation
    Test-Installation -BinaryPath $binaryPath
}

# Check if running with appropriate permissions
if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Info "Running as regular user (recommended)"
}

# Run main function
try {
    Main
}
catch {
    Write-Error "Installation failed: $($_.Exception.Message)"
}