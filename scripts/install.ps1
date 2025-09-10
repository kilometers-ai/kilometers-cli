# Kilometers CLI installation script for Windows
# Usage: irm https://raw.githubusercontent.com/kilometers-ai/kilometers-cli/main/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

$REPO = "kilometers-ai/kilometers-cli"
$BINARY_NAME = "km"

Write-Host "Kilometers CLI Installer" -ForegroundColor Green
Write-Host "========================" -ForegroundColor Green

# Detect architecture
$ARCH = if ([Environment]::Is64BitOperatingSystem) { "x86_64" } else { "i686" }
$PLATFORM = "$ARCH-pc-windows-msvc"

# Get latest release
Write-Host "Fetching latest release..." -ForegroundColor Yellow
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"
    $version = $release.tag_name
} catch {
    Write-Host "Failed to fetch latest release: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Latest release: $version" -ForegroundColor Green

# Download URL
$downloadUrl = "https://github.com/$REPO/releases/download/$version/km-$PLATFORM.zip"

# Create temporary directory
$tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }

try {
    # Download binary
    Write-Host "Downloading $BINARY_NAME for Windows ($ARCH)..." -ForegroundColor Yellow
    $zipPath = Join-Path $tempDir "km.zip"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing

    # Extract binary
    Write-Host "Extracting binary..." -ForegroundColor Yellow
    Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force

    # Check if binary exists
    $binaryPath = Join-Path $tempDir "km.exe"
    if (!(Test-Path $binaryPath)) {
        throw "Binary not found in archive"
    }

    # Install directory
    $installDir = "$env:LOCALAPPDATA\Programs\km"
    if (!(Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    # Move binary to install directory
    $installedBinary = Join-Path $installDir "km.exe"
    Write-Host "Installing $BINARY_NAME to $installDir..." -ForegroundColor Yellow
    Move-Item -Path $binaryPath -Destination $installedBinary -Force

    # Add to PATH if not already there
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$installDir*") {
        Write-Host "Adding $installDir to PATH..." -ForegroundColor Yellow
        $newPath = "$userPath;$installDir"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$env:Path;$installDir"
        Write-Host "PATH updated. You may need to restart your terminal." -ForegroundColor Yellow
    }

    # Verify installation
    Write-Host "`n✓ Kilometers CLI installed successfully!" -ForegroundColor Green
    Write-Host "Version: $version"

    Write-Host "`nGet started with:" -ForegroundColor Cyan
    Write-Host "  km init        # Initialize with your API key"
    Write-Host "  km --help      # Show available commands"

} catch {
    Write-Host "Installation failed: $_" -ForegroundColor Red
    exit 1
} finally {
    # Cleanup
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force $tempDir
    }
}