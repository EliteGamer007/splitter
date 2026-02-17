# Run Instance 2 of Splitter (Federation Testing)
# Uses .env.instance2 for separate database, port, and federation domain

Write-Host "Starting Splitter Instance 2 (splitter-2) on port 8001..." -ForegroundColor Cyan
Write-Host "Database: neondb_2 | Domain: splitter-2 | URL: http://localhost:8001" -ForegroundColor Yellow

# Load env vars from .env.instance2
Get-Content ".env.instance2" | ForEach-Object {
    if ($_ -match "^\s*#" -or $_ -match "^\s*$") { return }
    $parts = $_ -split "=", 2
    if ($parts.Count -eq 2) {
        $key = $parts[0].Trim()
        $value = $parts[1].Trim()
        [Environment]::SetEnvironmentVariable($key, $value, "Process")
    }
}

Write-Host "`nEnvironment loaded. Starting server..." -ForegroundColor Green
go run cmd/server/main.go
