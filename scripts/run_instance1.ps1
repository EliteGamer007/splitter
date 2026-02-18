# Start Instance 1 (splitter-1) on port 8000
# Uses default .env file

Write-Host "Starting Splitter Instance 1 (splitter-1) on port 8000..." -ForegroundColor Cyan
Write-Host "Database: neondb | Domain: splitter-1 | URL: http://localhost:8000" -ForegroundColor Yellow
Write-Host ""

go run cmd/server/main.go
