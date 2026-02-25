# SLM Manager Windows Installer
# Run as: .\install-windows.ps1

param(
    [string]$InstallDir = "C:\slm-manager",
    [switch]$NoStartup = $false
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 SLM Manager Installer" -ForegroundColor Cyan
Write-Host "========================" -ForegroundColor Cyan
Write-Host ""

# Create installation directory
if (-not (Test-Path $InstallDir)) {
    Write-Host "Creating installation directory: $InstallDir"
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Download latest release from GitHub
$repo = "kooshapari/bifrost-extensions"
$releaseUrl = "https://api.github.com/repos/$repo/releases/latest"

try {
    Write-Host "Checking for latest release..."
    $release = Invoke-RestMethod -Uri $releaseUrl -Headers @{"Accept"="application/vnd.github.v3+json"}
    $version = $release.tag_name
    Write-Host "Latest version: $version" -ForegroundColor Green
    
    # Find Windows asset
    $asset = $release.assets | Where-Object { $_.name -like "*windows*amd64*.zip" }
    if (-not $asset) {
        throw "No Windows release found"
    }
    
    $downloadUrl = $asset.browser_download_url
    $zipPath = "$env:TEMP\slm-manager.zip"
    
    Write-Host "Downloading $($asset.name)..."
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath
    
    Write-Host "Extracting..."
    Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force
    Remove-Item $zipPath
    
} catch {
    Write-Host "Could not download from GitHub. Using local files if available..." -ForegroundColor Yellow
    
    # Copy local files if they exist
    $localExe = Join-Path $PSScriptRoot "..\slm-manager.exe"
    $localSlmServer = Join-Path $PSScriptRoot "..\slm-server.exe"
    
    if (Test-Path $localExe) {
        Copy-Item $localExe $InstallDir
    }
    if (Test-Path $localSlmServer) {
        Copy-Item $localSlmServer $InstallDir
    }
}

# Create start script
$startScript = @"
@echo off
cd /d "$InstallDir"
start "" slm-manager.exe
"@
Set-Content -Path "$InstallDir\start.bat" -Value $startScript

# Add to startup (optional)
if (-not $NoStartup) {
    Write-Host "Adding to Windows startup..."
    $startupPath = [Environment]::GetFolderPath("Startup")
    $shortcutPath = Join-Path $startupPath "SLM Manager.lnk"
    
    $shell = New-Object -ComObject WScript.Shell
    $shortcut = $shell.CreateShortcut($shortcutPath)
    $shortcut.TargetPath = "$InstallDir\slm-manager.exe"
    $shortcut.WorkingDirectory = $InstallDir
    $shortcut.Description = "SLM Manager - Manages SLM routing server and vLLM"
    $shortcut.Save()
    
    Write-Host "Added startup shortcut" -ForegroundColor Green
}

# Setup WSL vLLM (if WSL is available)
$wslCheck = Get-Command wsl -ErrorAction SilentlyContinue
if ($wslCheck) {
    Write-Host ""
    Write-Host "WSL detected. Setting up vLLM environment..." -ForegroundColor Cyan
    
    $setupScript = @'
#!/bin/bash
set -e

# Create Python virtual environment
python3 -m venv ~/vllm-env
source ~/vllm-env/bin/activate

# Install vLLM
pip install --upgrade pip
pip install vllm

# Create start script
cat > ~/start-vllm.sh << 'EOF'
#!/bin/bash
source ~/vllm-env/bin/activate
vllm serve Qwen/Qwen2.5-3B-Instruct --port 8000 --host 0.0.0.0
EOF
chmod +x ~/start-vllm.sh

echo "vLLM setup complete!"
'@
    
    $setupPath = "$env:TEMP\setup-vllm.sh"
    Set-Content -Path $setupPath -Value $setupScript -NoNewline
    
    Write-Host "Running vLLM setup in WSL (this may take a while)..."
    wsl bash $setupPath
}

Write-Host ""
Write-Host "✅ Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "To start SLM Manager:"
Write-Host "  $InstallDir\start.bat" -ForegroundColor Yellow
Write-Host ""
Write-Host "The app will also start automatically on login." -ForegroundColor Gray

