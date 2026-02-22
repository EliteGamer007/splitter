# Apply Federation Migration to Both Instances
# This script applies the federation fix migration to both databases

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Applying Federation Migration" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Load instance 1 env
$env1 = Get-Content .env | Where-Object { $_ -notmatch '^#' -and $_ -match '=' }
$envVars1 = @{}
foreach ($line in $env1) {
    $parts = $line -split '=', 2
    if ($parts.Length -eq 2) {
        $envVars1[$parts[0].Trim()] = $parts[1].Trim()
    }
}

# Build instance 2 env from .env INSTANCE2_* overrides
$envVars2 = @{}
$envVars2['DB_HOST'] = $envVars1['DB_HOST']
$envVars2['DB_PORT'] = $envVars1['DB_PORT']
$envVars2['DB_USER'] = $envVars1['DB_USER']
$envVars2['DB_PASSWORD'] = $envVars1['DB_PASSWORD']
$envVars2['DB_NAME'] = $envVars1['INSTANCE2_DB_NAME']

# Function to apply migration
function Apply-Migration {
    param (
        [string]$dbHost,
        [string]$dbPort,
        [string]$dbUser,
        [string]$dbPassword,
        [string]$dbName,
        [string]$instanceName
    )
    
    Write-Host "Applying migration to $instanceName ($dbName)..." -ForegroundColor Yellow
    
    $env:PGPASSWORD = $dbPassword
    $psqlCmd = "psql -h $dbHost -p $dbPort -U $dbUser -d $dbName -f migrations/002_upgrade_to_current.sql"
    
    try {
        Invoke-Expression $psqlCmd
        Write-Host "✓ Migration applied successfully to $instanceName" -ForegroundColor Green
    } catch {
        Write-Host "✗ Failed to apply migration to $instanceName : $_" -ForegroundColor Red
    }
    
    Remove-Item Env:PGPASSWORD
}

# Apply to Instance 1
Write-Host ""
Apply-Migration -dbHost $envVars1['DB_HOST'] `
                -dbPort $envVars1['DB_PORT'] `
                -dbUser $envVars1['DB_USER'] `
                -dbPassword $envVars1['DB_PASSWORD'] `
                -dbName $envVars1['DB_NAME'] `
                -instanceName "Instance 1 (splitter-1)"

Write-Host ""

# Apply to Instance 2
Apply-Migration -dbHost $envVars2['DB_HOST'] `
                -dbPort $envVars2['DB_PORT'] `
                -dbUser $envVars2['DB_USER'] `
                -dbPassword $envVars2['DB_PASSWORD'] `
                -dbName $envVars2['DB_NAME'] `
                -instanceName "Instance 2 (splitter-2)"

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Migration Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
