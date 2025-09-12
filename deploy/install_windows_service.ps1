param(
  [string]$ServiceName = "Wartracker",
  [string]$NssmPath = "nssm",
  [string]$BotToken = $env:BOT_TOKEN,
  [string]$DbPath = "$PSScriptRoot\..\data\guild_data.db",
  [string]$ExePath = "$PSScriptRoot\..\bot.exe",
  [string]$WorkingDir = "$PSScriptRoot\.."
)

Write-Host "Installing Windows service '$ServiceName' using NSSM..."

if (-not (Get-Command $NssmPath -ErrorAction SilentlyContinue)) {
  Write-Error "NSSM not found. Install NSSM and ensure 'nssm' is on PATH, or pass -NssmPath with full path to nssm.exe."
  exit 1
}
if (-not (Test-Path $ExePath)) {
  Write-Error "Executable not found at $ExePath. Build it first: 'go build -o bot.exe ./cmd/bot'"
  exit 1
}
if (-not $BotToken) {
  Write-Error "Bot token not provided. Pass -BotToken or set BOT_TOKEN env."
  exit 1
}

# Ensure data directory exists
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $DbPath) | Out-Null

# Create or update service
& $NssmPath install $ServiceName $ExePath | Out-Null
& $NssmPath set $ServiceName AppDirectory $WorkingDir | Out-Null
& $NssmPath set $ServiceName AppStopMethodSkip 0 | Out-Null
& $NssmPath set $ServiceName AppRestartDelay 3000 | Out-Null
& $NssmPath set $ServiceName Start SERVICE_AUTO_START | Out-Null

# Set environment variables per-service (no need to alter system env)
$envList = @(
  "BOT_TOKEN=$BotToken",
  "DB_PATH=$DbPath",
  "ENABLE_GUILD_MEMBERS_INTENT=1"
)
& $NssmPath set $ServiceName AppEnvironmentExtra ($envList -join " ") | Out-Null

# Start service
& $NssmPath start $ServiceName | Out-Null
Write-Host "Service '$ServiceName' installed and started. Use 'nssm edit $ServiceName' to adjust settings."
