$ErrorActionPreference = "Continue"

$resultsDir = "tests\results"
if (-Not (Test-Path -Path $resultsDir)) {
    New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null
}

$resultsFile = "$resultsDir\regression_results.txt"

$header = "Starting Splitter Regression Test Suite..."
Write-Host $header -ForegroundColor Cyan
"--- Splitter Regression Test Suite ---" | Out-File -FilePath $resultsFile -Encoding UTF8
"Started at: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')`n" | Out-File -FilePath $resultsFile -Append -Encoding UTF8

$testCategories = [ordered]@{
    "Unit"            = "./tests/unit/..."
    "Database"        = "./tests/db"
    "Schema"          = "./tests/check_schema"
    "User Validation" = "./tests/check_users"
    "Encryption"      = "./tests/check_encryption_keys"
    "Migration"       = "./tests/apply_migration"
    "Seeder"          = "./tests/seeder"
    "Verify DB"       = "./tests/verify_db"
    "Fix DB"          = "./tests/fix_db"
    "Integration"     = "./tests/integration"
    "E2E"             = "./tests/e2e_test"
    "Load"            = "./tests/load"
}

foreach ($key in $testCategories.Keys) {
    $path = $testCategories[$key]
    $ts = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $msg = "[$ts] Running $key tests ($path)..."
    
    Write-Host $msg -ForegroundColor Yellow
    $msg | Out-File -FilePath $resultsFile -Append -Encoding UTF8
    
    # Run the tests
    $output = go test $path 2>&1
    $exitCode = $LASTEXITCODE

    foreach ($line in $output) {
        Write-Host $line
        $line | Out-File -FilePath $resultsFile -Append -Encoding UTF8
    }

    $endTs = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    if ($exitCode -eq 0) {
        $statusMsg = "[$endTs] $key tests PASSED.`n"
        Write-Host $statusMsg -ForegroundColor Green
        $statusMsg | Out-File -FilePath $resultsFile -Append -Encoding UTF8
    } else {
        $statusMsg = "[$endTs] $key tests FAILED.`n"
        Write-Host $statusMsg -ForegroundColor Red
        $statusMsg | Out-File -FilePath $resultsFile -Append -Encoding UTF8
    }
}

$completionMsg = "Splitter Regression Test Suite Completed at $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
Write-Host $completionMsg -ForegroundColor Cyan
$completionMsg | Out-File -FilePath $resultsFile -Append -Encoding UTF8
