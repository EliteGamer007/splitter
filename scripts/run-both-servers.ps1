Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = "c:\Users\Sanjeev Srinivas\Desktop\splitter"

Write-Host "Starting Splitter backend instance 1 (port 8000)..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList @(
  "-NoExit",
  "-Command",
  "Set-Location '$repoRoot'; `$env:ENV_FILE='.env'; `$env:PORT='8000'; `$env:FEDERATION_DOMAIN='splitter-1'; `$env:FEDERATION_URL='http://localhost:8000'; `$env:FEDERATION_ENABLED='true'; go run cmd/server/main.go"
)

Write-Host "Starting Splitter backend instance 2 (port 8001)..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList @(
  "-NoExit",
  "-Command",
  "Set-Location '$repoRoot'; `$env:ENV_FILE='.env'; `$env:DB_NAME='neondb_2'; `$env:PORT='8001'; `$env:BASE_URL='http://localhost:8001'; `$env:JWT_SECRET='instance-2-jwt-secret-key'; `$env:FEDERATION_DOMAIN='splitter-2'; `$env:FEDERATION_URL='http://localhost:8001'; `$env:FEDERATION_ENABLED='true'; go run cmd/server/main.go"
)

Write-Host "Both server terminals launched." -ForegroundColor Green