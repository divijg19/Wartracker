# PowerShell helper to run the bot locally with a persistent data folder
param(
    [string]$BotToken = $env:BOT_TOKEN,
    [string]$DbPath = "$PSScriptRoot\..\data\guild_data.db"
)
if (-not $BotToken) {
    Write-Host "Set BOT_TOKEN env or pass -BotToken"
    exit 1
}

$env:BOT_TOKEN = $BotToken
$env:DB_PATH = $DbPath

# Ensure data dir exists
New-Item -ItemType Directory -Force -Path (Split-Path $DbPath)

# Run
& dotnet --version > $null 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "Note: dotnet present — ensure this is intended."
}

Start-Process -NoNewWindow -FilePath "go" -ArgumentList 'run','./cmd/bot' -Wait
