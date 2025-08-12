# Odyssey Wallet Windows Installer
# This script downloads and installs the latest version of Odyssey Wallet

param(
    [string]$Version = "latest",
    [string]$InstallDir = "",
    [switch]$Force = $false
)

$ErrorActionPreference = 'Stop'

# Colors for output
function Write-ColorOutput($ForegroundColor, $Message) {
    Write-Host $Message -ForegroundColor $ForegroundColor
}

function Write-Success($Message) { Write-ColorOutput Green $Message }
function Write-Error($Message) { Write-ColorOutput Red $Message }
function Write-Warning($Message) { Write-ColorOutput Yellow $Message }
function Write-Info($Message) { Write-ColorOutput Cyan $Message }

function Get-GoInstallPath {
    # Try to find Go installation from registry
    $goPaths = @(
        "HKLM:\SOFTWARE\Go",
        "HKLM:\SOFTWARE\WOW6432Node\Go",
        "HKCU:\SOFTWARE\Go"
    )
    
    foreach ($path in $goPaths) {
        try {
            $goInfo = Get-ItemProperty -Path $path -ErrorAction SilentlyContinue
            if ($goInfo -and $goInfo.InstallDir) {
                $goBinPath = Join-Path $goInfo.InstallDir "bin"
                if (Test-Path $goBinPath) {
                    return $goBinPath
                }
            }
        } catch {
            # Continue to next registry path
        }
    }
    
    # Fallback: check common installation paths
    $commonPaths = @(
        "$env:ProgramFiles\Go\bin",
        "${env:ProgramFiles(x86)}\Go\bin",
        "$env:USERPROFILE\go\bin",
        "$env:LOCALAPPDATA\go\bin"
    )
    
    foreach ($path in $commonPaths) {
        if (Test-Path $path) {
            return $path
        }
    }
    
    return $null
}

function Test-GitInstalled {
    try {
        $null = git --version 2>$null
        return $true
    } catch {
        return $false
    }
}

function Show-GitInstallInstructions {
    Write-Error "Git is required for Go module dependencies but is not installed."
    Write-Host ""
    Write-Info "📥 How to install Git:"
    Write-Host ""
    Write-Info "Option 1 - Download from official website:"
    Write-Host "  1. Visit: https://git-scm.com/download/win" -ForegroundColor Cyan
    Write-Host "  2. Download the installer for your system" -ForegroundColor Cyan
    Write-Host "  3. Run the installer with default settings" -ForegroundColor Cyan
    Write-Host ""
    Write-Info "Option 2 - Using Package Managers:"
    Write-Host "  # Using Chocolatey:" -ForegroundColor Cyan
    Write-Host "  choco install git" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  # Using Winget:" -ForegroundColor Cyan
    Write-Host "  winget install --id Git.Git -e --source winget" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  # Using Scoop:" -ForegroundColor Cyan
    Write-Host "  scoop install git" -ForegroundColor Yellow
    Write-Host ""
    Write-Warning "After installing Git, restart your terminal and run this script again."
}

function Show-GoInstallInstructions {
    $is64Bit = [System.Environment]::Is64BitOperatingSystem
    $goArch = if ($arch -eq "386") { "386" } elseif ($arch -eq "arm64") { "arm64" } else { "amd64" }
    
    Write-Error "Go programming language is required but is not installed."
    Write-Host ""
    Write-Info "📥 How to install Go:"
    Write-Host ""
    Write-Info "Option 1 - Download from official website:"
    Write-Host "  1. Visit: https://golang.org/dl/" -ForegroundColor Cyan
    Write-Host "  2. Download Go for Windows $goArch" -ForegroundColor Cyan
    Write-Host "  3. Run the MSI installer" -ForegroundColor Cyan
    Write-Host "  4. Restart your terminal after installation" -ForegroundColor Cyan
    Write-Host ""
    Write-Info "Option 2 - Using Package Managers:"
    Write-Host "  # Using Chocolatey:" -ForegroundColor Cyan
    Write-Host "  choco install golang" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  # Using Winget:" -ForegroundColor Cyan
    Write-Host "  winget install --id GoLang.Go" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  # Using Scoop:" -ForegroundColor Cyan
    Write-Host "  scoop install go" -ForegroundColor Yellow
    Write-Host ""
    Write-Info "💡 Recommended: Use the official MSI installer for best compatibility"
    Write-Warning "After installing Go, restart your terminal and run this script again."
}

function Refresh-EnvironmentPath {
    # Refresh PATH for current session
    $machinePath = [System.Environment]::GetEnvironmentVariable("PATH", "Machine")
    $userPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
    $env:PATH = $machinePath + ";" + $userPath
}

Write-Info "🚀 Odyssey Wallet Windows Installer"
Write-Info "======================================"
Write-Host ""

# Check PowerShell version
if ($PSVersionTable.PSVersion.Major -lt 3) {
    Write-Error "PowerShell 3.0 or higher is required. Please upgrade PowerShell."
    exit 1
}

# Determine architecture using proper .NET method
$is64Bit = [System.Environment]::Is64BitOperatingSystem
$arch = if ($is64Bit) { "amd64" } else { "386" }

# Check for ARM64
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $arch = "arm64"
}

Write-Info "Detected architecture: $arch (64-bit OS: $is64Bit)"

# Set installation directory
if (!$InstallDir) {
    $InstallDir = "$env:LOCALAPPDATA\Programs\Odyssey"
}

Write-Info "Installation directory: $InstallDir"

# Create installation directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Get latest version if not specified
if ($Version -eq "latest") {
    Write-Info "Fetching latest version information..."
    try {
        $releaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/chinmay1088/odyssey/releases/latest" -UseBasicParsing
        $Version = $releaseInfo.tag_name
        Write-Info "Latest version: $Version"
    } catch {
        Write-Warning "Failed to fetch latest version, using main branch: $($_.Exception.Message)"
        $Version = "main"
    }
}

Write-Info "Installing Odyssey Wallet $Version"

# Check if Go is installed
$goFound = $false
try {
    $goVersion = go version 2>$null
    if ($goVersion) {
        $goFound = $true
        Write-Info "Found Go: $goVersion"
    }
} catch {
    # Go not found in PATH
}

if (-not $goFound) {
    Show-GoInstallInstructions
    exit 1
}

# Check if Git is installed (required for Go modules)
if (-not (Test-GitInstalled)) {
    $gitInstalled = Install-Git
    if (-not $gitInstalled) {
        Write-Error "Git installation failed. Cannot proceed with build."
        exit 1
    }
}

# Download source code
if ($Version -eq "latest" -or $Version -eq "main") {
    $sourceUrl = "https://github.com/chinmay1088/odyssey/archive/refs/heads/main.zip"
    Write-Warning "Building from main branch - this may be unstable"
} else {
    $sourceUrl = "https://github.com/chinmay1088/odyssey/archive/refs/tags/$Version.zip"
}

$zipFile = "$env:TEMP\odyssey-source.zip"

Write-Info "Downloading from: $sourceUrl"

try {
    # Download the source
    Invoke-WebRequest -Uri $sourceUrl -OutFile $zipFile -UseBasicParsing
    Write-Success "✅ Download completed"
} catch {
    if ($Version -ne "main") {
        Write-Warning "Failed to download tagged version, trying main branch..."
        $sourceUrl = "https://github.com/chinmay1088/odyssey/archive/refs/heads/main.zip"
        try {
            Invoke-WebRequest -Uri $sourceUrl -OutFile $zipFile -UseBasicParsing
            Write-Success "✅ Download completed from main branch"
        } catch {
            Write-Error "Failed to download Odyssey source: $($_.Exception.Message)"
            exit 1
        }
    } else {
        Write-Error "Failed to download Odyssey source: $($_.Exception.Message)"
        exit 1
    }
}

try {
    # Extract the source
    Write-Info "Extracting source code..."
    $tempSourceDir = "$env:TEMP\odyssey-source"
    if (Test-Path $tempSourceDir) {
        Remove-Item $tempSourceDir -Recurse -Force
    }
    Expand-Archive -Path $zipFile -DestinationPath $tempSourceDir -Force
    
    # Find the source directory (GitHub creates a folder like odyssey-main or odyssey-v1.0.0)
    $sourceDir = Get-ChildItem -Path $tempSourceDir -Directory | Select-Object -First 1
    if (!$sourceDir) {
        Write-Error "Source directory not found"
        exit 1
    }
    
    Write-Info "Building Odyssey..."
    Write-Info "Source directory: $($sourceDir.FullName)"
    
    Push-Location $sourceDir.FullName
    
    # Initialize go modules if go.mod doesn't exist
    if (!(Test-Path "go.mod")) {
        Write-Info "Initializing Go modules..."
        go mod init odyssey
        go mod tidy
    }
    
    # Download dependencies
    Write-Info "Downloading Go dependencies..."
    go mod download
    
    # Build the binary
    $env:CGO_ENABLED = "0"
    Write-Info "Compiling Odyssey binary..."
    go build -ldflags "-s -w" -o odyssey.exe .
    
    if (!(Test-Path "odyssey.exe")) {
        Write-Error "Build failed - odyssey.exe not created"
        Write-Info "Trying alternative build command..."
        go build -o odyssey.exe ./cmd/...
        
        if (!(Test-Path "odyssey.exe")) {
            Write-Error "Alternative build also failed"
            exit 1
        }
    }
    
    Pop-Location
    
    # Copy to install directory
    Copy-Item "$($sourceDir.FullName)\odyssey.exe" $InstallDir -Force
    
    $exePath = Join-Path $InstallDir "odyssey.exe"
    Write-Success "✅ Build and installation completed"
    
} catch {
    Write-Error "Failed to build Odyssey: $($_.Exception.Message)"
    Write-Error "Build output: $($Error[0].Exception.Message)"
    exit 1
} finally {
    # Clean up
    if (Test-Path $zipFile) {
        Remove-Item $zipFile -Force
    }
    if (Test-Path "$env:TEMP\odyssey-source") {
        Remove-Item "$env:TEMP\odyssey-source" -Recurse -Force
    }
}

# Add to PATH
$currentUserPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($currentUserPath -notlike "*$InstallDir*") {
    Write-Info "Adding Odyssey to your PATH..."
    $newPath = if ($currentUserPath) { "$currentUserPath;$InstallDir" } else { $InstallDir }
    [System.Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    Write-Success "✅ Added to PATH"
    
    # Update current session PATH
    Refresh-EnvironmentPath
    
    # Test if odyssey command works in current session
    try {
        $testPath = Get-Command odyssey -ErrorAction SilentlyContinue
        if ($testPath) {
            Write-Success "✅ Odyssey command available in current session"
        } else {
            Write-Warning "⚠️  Odyssey added to PATH but not available in current session"
            Write-Warning "    Please restart your terminal or run: refreshenv"
        }
    } catch {
        Write-Warning "⚠️  PATH updated but command test failed"
    }
} else {
    Write-Info "Odyssey directory already in PATH"
    Refresh-EnvironmentPath
}

# Verify installation
Write-Info "Verifying installation..."
try {
    $versionOutput = & $exePath version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "✅ Installation verified: $versionOutput"
    } else {
        # Try alternative version commands
        $altVersionOutput = & $exePath --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "✅ Installation verified: $altVersionOutput"
        } else {
            Write-Success "✅ Installation completed (executable created successfully)"
            Write-Info "Note: Version command may not be implemented yet"
        }
    }
} catch {
    Write-Success "✅ Installation completed (executable created successfully)"
    Write-Warning "⚠️  Version verification skipped: $($_.Exception.Message)"
}

Write-Host ""
Write-Success "🎉 Odyssey Wallet installed successfully!"
Write-Host ""
Write-Info "Installation Details:"
Write-Host "  Executable: $exePath" -ForegroundColor White
Write-Host "  Version: $Version" -ForegroundColor White
Write-Host ""
Write-Info "Quick Start:"
Write-Host "  odyssey init      # Create a new wallet" -ForegroundColor Cyan
Write-Host "  odyssey unlock    # Unlock your wallet" -ForegroundColor Cyan
Write-Host "  odyssey balance   # Check your balances" -ForegroundColor Cyan
Write-Host "  odyssey --help    # Show all commands" -ForegroundColor Cyan
Write-Host ""
Write-Info "Documentation: https://github.com/chinmay1088/odyssey/blob/main/README.MD"
Write-Host ""
Write-Warning "⚠️  IMPORTANT SECURITY NOTES:"
Write-Warning "• Your wallet will be stored in: $env:USERPROFILE\.odyssey\"
Write-Warning "• Always backup your recovery phrase"
Write-Warning "• Never share your recovery phrase with anyone"
Write-Warning "• Keep your password secure"
Write-Host ""

# Final PATH reminder
$pathWarning = $false
try {
    $testCmd = Get-Command odyssey -ErrorAction SilentlyContinue
    if (-not $testCmd) {
        $pathWarning = $true
    }
} catch {
    $pathWarning = $true
}

if ($pathWarning) {
    Write-Warning "💡 Important: You may need to restart your terminal/PowerShell to use the 'odyssey' command"
    Write-Info "    Or run this command to refresh your current session:"
    Write-Host "    `$env:PATH = [System.Environment]::GetEnvironmentVariable('PATH', 'User') + ';' + [System.Environment]::GetEnvironmentVariable('PATH', 'Machine')" -ForegroundColor Yellow
} else {
    Write-Success "✅ Odyssey command ready to use in current session!"
}