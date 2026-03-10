# Test Reporting Guide

## Overview
Test reports give stakeholders and developers a clear view into code coverage, execution success, and performance regressions. They are tracked and output into the `tests/reports/` directory.

## Generating Line Coverage
A critical metric is test coverage. You can generate HTML test reports natively with Go:
```bash
make test-cover
```
This runs the suite with `-coverprofile=coverage.out` and opens it in your default browser using `go tool cover -html=coverage.out`.

## JUnit and CI Test Output
If you are generating XML reports for CI parsing (like GitHub Actions test reporter), you might use extensions like `go-junit-report`.
```bash
go test -v ./... 2>&1 | go-junit-report > tests/reports/report.xml
```

## Performance Testing Reports
When utilizing `k6`, graphical reports or JSON metrics will also be dumped into `tests/reports/`.
Review these results closely after making significant algorithm or database query adjustments to ensure no unintended throughput degradation.
