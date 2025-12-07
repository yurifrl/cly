# NSX installer for Windows
#
# Usage:
#   iwr -useb https://nsx-cli-proxy.nsx.services/main/install.ps1 | iex
#   iwr -useb https://nsx-cli-proxy.nsx.services/main/install.ps1 | iex -Args "-Version v1.0.0"
#
# Parameters:
#   -Version: Override the default version to install
#   -InstallDir: Override the default installation directory

param (
    [string]$Version = "latest",
    [string]$InstallDir = ""
)

$ErrorActionPreference = 'Stop'

# Check if running with admin privileges
function Test-Admin {
    $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    return $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Write-Info {
    param([string]$Message)
    Write-Host "INFO: $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "WARN: $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "ERROR: $Message" -ForegroundColor Red
}

# Set default installation directory if not provided
if ([string]::IsNullOrEmpty($InstallDir)) {
    if (Test-Admin) {
        $InstallDir = "$env:ProgramFiles\nsx"
    } else {
        $InstallDir = "$env:LOCALAPPDATA\nsx"
    }
}

# Determine architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "x86_64" } else { "386" }
Write-Info "Detected Windows $arch architecture"

# Get latest version if not specified
if ($Version -eq "latest") {
    Write-Info "Finding latest version..."
    try {
        $latestRelease = Invoke-RestMethod -Uri "https://nsx-cli-proxy.nsx.services/latest"
        $Version = $latestRelease.tag_name
        Write-Info "Latest version is $Version"
    } catch {
        Write-Error "Failed to find latest version. Please specify a version manually."
        exit 1
    }
}

# Create temporary directory
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir | Out-Null
Write-Info "Created temporary directory: $tempDir"

try {
    # Build download URL
    $fileName = "nsx_Windows_$arch.zip"
    $downloadUrl = "https://nsx-cli-proxy.nsx.services/releases/download/$Version/$fileName"
    $zipFile = Join-Path $tempDir $fileName

    # Download the zip file
    Write-Info "Downloading $downloadUrl..."
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipFile

    # Create installation directory if it doesn't exist
    if (-not (Test-Path $InstallDir)) {
        Write-Info "Creating installation directory: $InstallDir"
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    # Extract the zip file
    Write-Info "Extracting files to $InstallDir..."
    Expand-Archive -Path $zipFile -DestinationPath $tempDir -Force
    Copy-Item -Path "$tempDir\nsx.exe" -Destination "$InstallDir\nsx.exe" -Force

    # Add to PATH if running as admin and not already in PATH
    if (Test-Admin) {
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
        if ($currentPath -notlike "*$InstallDir*") {
            Write-Info "Adding to system PATH..."
            [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$InstallDir", "Machine")
            $env:PATH = "$env:PATH;$InstallDir"
        }
    } else {
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($currentPath -notlike "*$InstallDir*") {
            Write-Info "Adding to user PATH..."
            [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$InstallDir", "User")
            $env:PATH = "$env:PATH;$InstallDir"
        }
    }

    Write-Info "Successfully installed NSX $Version to $InstallDir"
    Write-Info "Run 'nsx --help' to get started"

    # Set up shell completion
    Write-Info "To enable PowerShell tab completion, run:"
    Write-Host "  nsx completion powershell >> `$PROFILE"

    # Notify user if installation won't be available in current shell
    if (-not (Get-Command "nsx" -ErrorAction SilentlyContinue)) {
        Write-Warning "Please restart your terminal or run 'refreshenv' to use nsx from the command line"
    }
} catch {
    Write-Error "Installation failed: $_"
    exit 1
} finally {
    # Clean up temporary directory
    if (Test-Path $tempDir) {
        Write-Info "Cleaning up temporary files..."
        Remove-Item -Recurse -Force $tempDir
    }
}
