# Load and Performance Testing Guide

## Overview
As the Splitter application handles user transactions, expense groups, and active settlements, it must remain performant under stress. Our load testing strategy is defined and maintained in the `tests/load/` directory.

## Testing Tools
Load testing is typically executed using external tools such as `k6` or `vegeta`. 
- Scripts defined in `tests/load/` are used to simulate concurrent users mapping to standard endpoints (e.g., login, create expense, list groups).

## How to Run Load Tests
From the root directory, you can trigger specific load testing scripts. Make sure the server instance you are testing against is fully initialized.

Example for `vegeta` or `k6` assuming a local deployment:
```bash
# Wait for the server to be ready, then trigger:
cd tests/load/
# e.g., run k6 script
k6 run script.js
```

## Performance Baselines
Ensure that average response times for read-heavy operations stay under 200ms at typical usage volumes. When modifying complex queries, run a performance check before proposing pull requests.

## Analyzing Results
After running, compare the generated latencies, P95, and P99 percentiles against previously established baselines to detect regressions.
