# l2c - One-line Installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/binodta/l2c/main/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "binodta/l2c"
$InstallDir = "$env:USERPROFILE\.l2c"
$BinaryName = "l2c.exe"
$Timestamp = [int][double]::Parse((Get-Date -UFormat %s))
$DownloadUrl = "https://raw.githubusercontent.com/$Repo/main/bin/l2c-windows-amd64.exe?v=$Timestamp"

Write-Host "Installing l2c (Local to Cloud)..." -ForegroundColor Cyan

# 1. Create install directory
if (Test-Path $InstallDir -PathType Leaf) {
    Remove-Item $InstallDir -Force
}
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# 2. Download the binary
Write-Host "Downloading binary for Windows (amd64)..."
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile "$InstallDir\$BinaryName" -UseBasicParsing
} catch {
    Write-Host "Error: Failed to download binary. Check your internet connection." -ForegroundColor Red
    exit 1
}

Write-Host "Binary downloaded successfully." -ForegroundColor Green

# 3. Add to PATH for the current user
$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to your PATH." -ForegroundColor Green
    Write-Host "Please restart your terminal for the PATH change to take effect."
} else {
    Write-Host "PATH already configured." -ForegroundColor Blue
}

# 4. Run setup
Write-Host "`nStarting l2c setup...`n" -ForegroundColor Cyan
& "$InstallDir\$BinaryName" setup

Write-Host "`nInstallation complete!" -ForegroundColor Green
Write-Host "------------------------------------------------"
Write-Host "You can now run: l2c run" -ForegroundColor Green
Write-Host "------------------------------------------------"
