Set-Location "c:\Users\Sanjeev Srinivas\Desktop\splitter"

Write-Host "Running recent implementation validation suite..." -ForegroundColor Cyan

go test ./internal/...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

go test ./tests/unit/...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

go test ./tests/integration/...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

go test ./tests/db/... ./tests/unit/... ./tests/integration/...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Recent implementation validation suite passed." -ForegroundColor Green
