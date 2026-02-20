# Run Instance 2 of Splitter (Federation Testing)
# Uses .env.instance2 for separate database, port, and federation domain

Write-Host "Starting Splitter Instance 2 (splitter-2) on port 8001..." -ForegroundColor Cyan
Write-Host "Database: neondb_2 | Domain: splitter-2 | URL: http://localhost:8001" -ForegroundColor Yellow

# Set ENV_FILE so config.go loads .env.instance2 instead of .env
$env:ENV_FILE = ".env.instance2"

Write-Host "`nEnvironment loaded. Starting server..." -ForegroundColor Green
Write-Host "DEBUG: ENV_FILE=$env:ENV_FILE" -ForegroundColor Magenta

go run cmd/server/main.go
