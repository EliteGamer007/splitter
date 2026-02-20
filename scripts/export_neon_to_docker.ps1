# Export Neon Data to Docker PostgreSQL
# Usage: Run this script to dump your Neon databases for import into Docker PostgreSQL

param(
    [string]$NeonHost = "ep-falling-mode-a1k832j8-pooler.ap-southeast-1.aws.neon.tech",
    [string]$NeonUser = "neondb_owner",
    [string]$NeonPassword = "npg_doQ6W7BuhytJ",
    [string]$OutputDir = ".\data_export"
)

Write-Host "=== Splitter: Neon to Docker Data Export ===" -ForegroundColor Cyan

# Create output directory
if (-Not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

# Set PGPASSWORD for non-interactive usage
$env:PGPASSWORD = $NeonPassword

Write-Host "`n1. Exporting neondb (Instance 1)..." -ForegroundColor Yellow
pg_dump --host=$NeonHost --port=5432 --username=$NeonUser --dbname=neondb --format=custom --file="$OutputDir\neondb_instance1.dump" --no-owner --no-privileges 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✅ neondb exported to $OutputDir\neondb_instance1.dump" -ForegroundColor Green
} else {
    Write-Host "   ⚠️ Export failed. Make sure pg_dump is installed (comes with PostgreSQL)" -ForegroundColor Red
    Write-Host "   Install: winget install PostgreSQL.PostgreSQL" -ForegroundColor Gray
}

Write-Host "`n2. Exporting neondb_2 (Instance 2)..." -ForegroundColor Yellow
pg_dump --host=$NeonHost --port=5432 --username=$NeonUser --dbname=neondb_2 --format=custom --file="$OutputDir\neondb_instance2.dump" --no-owner --no-privileges 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✅ neondb_2 exported to $OutputDir\neondb_instance2.dump" -ForegroundColor Green
} else {
    Write-Host "   ⚠️ Export failed" -ForegroundColor Red
}

# Clean up
Remove-Item Env:\PGPASSWORD -ErrorAction SilentlyContinue

Write-Host "`n=== Import into Docker PostgreSQL ===" -ForegroundColor Cyan
Write-Host "After starting Docker containers with PostgreSQL:" -ForegroundColor Gray
Write-Host "  docker exec -i splitter-db-1 pg_restore -U splitter_user -d splitter_db < $OutputDir\neondb_instance1.dump"
Write-Host "  docker exec -i splitter-db-2 pg_restore -U splitter_user -d splitter_db < $OutputDir\neondb_instance2.dump"
Write-Host ""
