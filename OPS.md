# Operations & Scaling Guide (OPS.md)

This guide covers the technical operations, monitoring, and scaling strategies for running a Splitter instance in production.

## Monitoring Strategy

### 1. Application Metrics (Go/Echo)
- **Prometheus**: The backend exports metrics on `:8080/metrics` (if configured). Key metrics to track:
    - `http_requests_total`: Request volume by code and method.
    - `go_goroutines`: Current goroutine count (indicator of leaks).
    - `db_open_connections`: Database connection pool health.
- **Grafana**: Use the standard "Go Metrics" dashboard to visualize these stats.

### 2. Database Monitoring (Neon.tech)
- Monitor **Auto-scaling events** in the Neon dashboard.
- Track **Active vs Idle sessions**.
- Set alerts for **Storage usage** approaching 80%.

### 3. Federation Health
- Check the **Federation Inspector** in the Admin UI for delivery failure rates and circuit breaker status.
- Monitor log levels for `[Federation]` tags to catch signature mismatches.

## Scaling Strategies

### Horizontal Scaling
- The Splitter backend is largely stateless (auth uses JWT).
- You can run multiple instances behind a Round Robin Load Balancer (e.g., Nginx, Render Load Balancer).
- **Sticky Sessions**: NOT required for API but recommended if using WebSockets for live notifications.

### Database Scaling
- Splitter uses **Neon.tech**, which scales compute automatically.
- For high-write loads, consider vertical scaling of the primary read/write node.
- Implement read-replicas if query latency increases significantly.

## Backup & Recovery

### Neon DB Backups
- Neon performs automatic maintenance backups.
- **Manual Export**:
    ```bash
    pg_dump -h db_host -U db_user db_name > backup.sql
    ```
- **Recovery**:
    ```bash
    psql -h db_host -U db_user db_name < backup.sql
    ```

## Environment Variable Reference

| Variable | Description | Example |
| --- | --- | --- |
| `PORT` | Backend listening port | `8000` |
| `DB_HOST` | Database host address | `ep-xyz.neon.tech` |
| `JWT_SECRET` | Signing key for tokens | `very-secret-string` |
| `FEDERATION_ENABLED` | Toggle AP federation | `true` |
| `LOG_LEVEL` | `info`, `debug`, `error` | `info` |
| `CORS_ORIGINS` | Allowed frontend domains | `https://app.splitter.social` |
