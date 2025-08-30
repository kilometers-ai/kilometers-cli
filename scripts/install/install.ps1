# Kilometers.ai CLI Installer for Windows
# Usage: iwr -useb https://get.kilometers.ai/install.ps1 | iex

param(
    [switch]$Uninstall,
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\Kilometers"
)

$ErrorActionPreference = "Stop"

# Configuration
$BinaryName = "km.exe"
$CDNBase = "https://get.kilometers.ai"
$GitHubRepo = "kilometers-ai/kilometers-cli"

# Helper functions
function Write-Info {
    Write-Host "[INFO]" -ForegroundColor Green -NoNewline
    Write-Host " $args"
}

function Write-Warn {
    Write-Host "[WARN]" -ForegroundColor Yellow -NoNewline
    Write-Host " $args"
}

function Write-Error {
    Write-Host "[ERROR]" -ForegroundColor Red -NoNewline
    Write-Host " $args" -ForegroundColor Red
    exit 1
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "x86" { return "386" }
        "ARM64" { return "arm64" }
        default { Write-Error "Unsupported architecture: $arch" }
    }
}

# Download binary
function Download-Binary {
    $arch = Get-Architecture
    $platform = "windows-$arch"
    $binaryFile = "km-$platform.exe"
    $folderName = "km-$platform"
    $downloadUrl = "$CDNBase/releases/latest/$folderName/$binaryFile"
    $tempFile = Join-Path $env:TEMP "km-download.exe"
    
    Write-Info "Downloading Kilometers CLI..."
    Write-Info "URL: $downloadUrl"
    
    try {
        # Download from CDN
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing
    }
    catch {
        Write-Error "Download failed. Please check your internet connection or try again later. Error: $_"
    }
    
    if (-not (Test-Path $tempFile) -or (Get-Item $tempFile).Length -eq 0) {
        Write-Error "Download failed or file is empty"
    }
    
    Write-Info "Download complete"
    return $tempFile
}

# Install binary
function Install-Binary {
    param($SourceFile)
    
    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    
    $targetPath = Join-Path $InstallDir $BinaryName
    
    Write-Info "Installing to $targetPath..."
    
    # Copy binary
    Copy-Item -Path $SourceFile -Destination $targetPath -Force
    
    # Add to PATH if not already there
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        Write-Info "Adding to PATH..."
        $newPath = "$userPath;$InstallDir"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$env:Path;$InstallDir"
    }
    
    # Clean up
    Remove-Item -Path $SourceFile -Force
    
    Write-Info "Installation successful!"
}

# Uninstall function
function Uninstall-Kilometers {
    Write-Info "Uninstalling Kilometers CLI..."
    
    $targetPath = Join-Path $InstallDir $BinaryName
    
    if (Test-Path $targetPath) {
        Remove-Item -Path $targetPath -Force
        Write-Info "Binary removed"
    }
    
    # Remove from PATH
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -like "*$InstallDir*") {
        $newPath = ($userPath -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Info "Removed from PATH"
    }
    
    # Remove directory if empty
    if (Test-Path $InstallDir) {
        $items = Get-ChildItem -Path $InstallDir
        if ($items.Count -eq 0) {
            Remove-Item -Path $InstallDir -Force
        }
    }
    
    Write-Info "Kilometers CLI uninstalled"
}

# Post-installation
function Show-PostInstall {
    # Test if km is available
    try {
        $version = & km --version 2>$null
        Write-Info "Installed version: $version"
    }
    catch {
        Write-Warn "Binary installed but not found in PATH"
        Write-Warn "You may need to restart your terminal"
    }
    
    Write-Host ""
    Write-Host "✓ Kilometers CLI installed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "To get started:"
    Write-Host "  1. Restart your terminal or run: `$env:Path = [Environment]::GetEnvironmentVariable('Path', 'User')"
    Write-Host "  2. Set your API key:"
    Write-Host "     `$env:KILOMETERS_API_KEY = 'your-api-key-here'"
    Write-Host "  3. Wrap your MCP server:"
    Write-Host "     km npx @modelcontextprotocol/server-github"
    Write-Host ""
    Write-Host "For more information:"
    Write-Host "  - Documentation: https://docs.kilometers.ai"
    Write-Host "  - Dashboard: https://app.kilometers.ai"
    Write-Host "  - Support: support@kilometers.ai"
    Write-Host ""
}

# Main
function Main {
    if ($Uninstall) {
        Uninstall-Kilometers
        return
    }
    
    Write-Info "Installing Kilometers CLI..."
    
    $tempFile = Download-Binary
    Install-Binary -SourceFile $tempFile
    Show-PostInstall
}

# Run
Main 