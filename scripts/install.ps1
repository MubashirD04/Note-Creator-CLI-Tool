# notes-cli Installer (Windows)
# This script builds and installs the notes-cli tool.

$ErrorActionPreference = "Stop"

if (!(Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Error: 'go' command not found. Please install Go (https://go.dev/dl/) first." -ForegroundColor Red
    exit 1
}

Write-Host "🚀 Building notes-cli..." -ForegroundColor Blue
go build -o notes-cli.exe

$installDir = "$HOME\bin"
if (!(Test-Path $installDir)) {
    Write-Host "📁 Creating $installDir..."
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Write-Host "📦 Copying binary to $installDir..." -ForegroundColor Blue
Copy-Item "notes-cli.exe" -Destination "$installDir\notes-cli.exe" -Force

# Check if $HOME\bin is in PATH
$path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($path -notlike "*$installDir*") {
    Write-Host "⚠️  $installDir is not in your PATH." -ForegroundColor Yellow
    Write-Host "Add it by running: [Environment]::SetEnvironmentVariable('Path', `"$path;$installDir`", 'User')" -ForegroundColor Yellow
}

Write-Host "⚙️  Initializing configuration..." -ForegroundColor Blue
$jsonPath = "$HOME\.notes-cli.json"
if (!(Test-Path $jsonPath)) {
    '{"courses": {}}' | Out-File -FilePath $jsonPath -Encoding utf8
}

Write-Host "✅ Installation complete!" -ForegroundColor Green
Write-Host "You may need to restart your terminal for 'notes-cli' to be recognized."
Write-Host "Launch the interactive wizard by running: notes-cli"
