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

# Load instance 2 env
$env2 = Get-Content .env.instance2 | Where-Object { $_ -notmatch '^#' -and $_ -match '=' }
$envVars2 = @{}
foreach ($line in $env2) {
    $parts = $line -split '=', 2
    if ($parts.Length -eq 2) {
        $envVars2[$parts[0].Trim()] = $parts[1].Trim()
    }
}

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
    $psqlCmd = "psql -h $dbHost -p $dbPort -U $dbUser -d $dbName -f migrations/010_federation_fix.sql"
    
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
