# Grafana Dashboards for Auth Service

Pre-built Grafana dashboards for monitoring the Auth Service, providing visibility into service performance, authentication operations, and infrastructure health.

## Available Dashboards

### 1. Auth Service Overview (`auth-service-overview.json`)

High-level overview of service health and performance with 9 panels:

- Request Rate by Endpoint
- P95 Latency
- Success Rate (%)
- Response Time Percentiles (P50, P95, P99)
- HTTP Status Codes Distribution
- In-Flight Requests
- Total Users
- Active Sessions
- Token Cache Hit Ratio

Use this for service health monitoring, SLA tracking, and quick status checks.

### 2. Authentication Metrics (`authentication-metrics.json`)

Deep dive into authentication operations and security with 13 panels:

- Login Success Rate
- Register Success Rate
- Token Refresh Success Rate
- Login Attempts (Success vs Failed)
- Register Attempts (Success vs Failed)
- OAuth Login Attempts by Provider
- 2FA Operations (Generated, Verified, Failed)
- Authentication Latency (P50, P95, P99)
- Rate Limit Violations by Endpoint
- Password Reset Requests
- Sessions Created
- Sessions Revoked
- Tokens Blacklisted

Use this for security monitoring, authentication troubleshooting, and fraud detection.

### 3. Infrastructure Metrics (`infrastructure-metrics.json`)

Database, cache, and business KPIs monitoring with 12 panels:

- Database Connection Pool Usage
- Database Query Latency (P50, P95, P99)
- Database Query Rate
- Redis Command Latency
- Redis Commands Rate
- Redis Connection Errors
- User Metrics (Total, Active, Growth)
- New User Registrations (24h)
- DB Connections Stats (Open, In-Use, Idle)
- Token Cache Hit Ratio
- Active Sessions Over Time

Use this for infrastructure health, resource forecasting, and business analytics.

## Quick Start

### Automatic Provisioning (Recommended)

When using Docker Compose, dashboards are automatically loaded:

```bash
# Start services with monitoring
docker-compose --profile monitoring up -d

# Access Grafana at http://localhost:3000
# Default credentials: admin / admin
```

Dashboards will be available under Dashboards → Browse → Auth Service folder.

### Manual Import

If you need to import manually:

1. Access Grafana at http://localhost:3000 (admin/admin)
2. Go to Dashboards → Import
3. Click Upload JSON file and select one of the dashboard files
4. Select Prometheus as the data source
5. Click Import
6. Repeat for each dashboard

## Dashboard Configuration

### Data Source

All dashboards use Prometheus as the data source. Ensure Prometheus is configured to scrape the Auth Service:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'auth-service'
    static_configs:
      - targets: ['auth-service:8080']
    metrics_path: '/metrics'
```

### Refresh Interval

Default is 10 seconds. You can change this in dashboard settings (top-right corner).

### Time Range

Default is last 1 hour. Quick ranges available: 5m, 15m, 1h, 6h, 24h, 7d, 30d. Custom range available in time picker.

## Customization

### Modifying Panels

To edit an existing panel:

1. Click dashboard title → Edit
2. Hover over the panel → Click Edit
3. Update the PromQL query
4. Click Save dashboard (top-right)

### Adding New Panels

To add a new panel:

1. Click Add → Visualization
2. Select a metric from available Prometheus metrics
3. Configure visualization type, thresholds, and colors
4. Click Apply

See `docs/PROMETHEUS_QUERIES.md` for 100+ PromQL queries covering HTTP requests, authentication, tokens, database, Redis, business KPIs, rate limiting, and email operations.

## Alerting

### Configuring Alerts

To create an alert:

1. Edit the panel with the metric you want to alert on
2. Click the Alert tab
3. Create an alert rule

Example: Alert when P95 latency exceeds 500ms:

```
Condition:
WHEN percentile(http_request_duration_seconds, 0.95) > 0.5
FOR 5 minutes

Notification:
- Slack channel: #alerts
- Email: ops@company.com
```

### Recommended Alerts

Critical alerts to configure:

- High Error Rate: Success rate < 95% for 5 minutes
- High Latency: P95 latency > 500ms for 5 minutes
- Database Issues: DB connection errors > 10 per minute
- Rate Limiting: Rate limit violations > 100 per minute

Warning alerts to consider:

- Cache Degradation: Cache hit ratio < 80%
- Slow Queries: P95 query latency > 100ms
- Session Growth: Active sessions growing > 20% in 1 hour
- Failed Logins: Failed login attempts > 50 per minute

## Troubleshooting

### Dashboard shows "No data"

Check the following:

1. Prometheus is running at http://localhost:9090
2. Prometheus is scraping Auth Service:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```
3. Auth Service is exposing metrics:
   ```bash
   curl http://localhost:8080/metrics
   ```
4. Data source is configured in Grafana: Configuration → Data Sources → Prometheus

### Some panels show "No data" but others work

This usually means:

1. The service hasn't received traffic yet (some metrics only appear after usage)
2. The metric name in the PromQL query is incorrect
3. The time range is too narrow (try "Last 24 hours")
4. Run the query directly in Prometheus at http://localhost:9090/graph to verify

### Grafana is using too much memory

To reduce memory usage:

1. Increase refresh interval from 10s to 30s or 1m
2. Reduce the time range in queries
3. Close unused dashboards
4. Increase Grafana memory limit in docker-compose.yml

## Dashboard Maintenance

### Exporting Dashboards

To save your customizations:

1. Open the dashboard
2. Click Dashboard Settings (gear icon) → JSON Model
3. Copy the JSON
4. Save to file: `auth-service-overview-custom.json`

### Version Control

Dashboard JSON files are version controlled. When updating:

1. Export the updated dashboard as JSON
2. Replace the existing file in `grafana/dashboards/`
3. Commit changes to git
4. Document changes in the commit message

### Best Practices

- Test changes in development before updating production dashboards
- Use variables for dynamic filtering (environment, region, etc.)
- Add descriptions to panels for context
- Organize panels logically (most important at the top)
- Use consistent colors across dashboards
- Set appropriate thresholds for visual indicators

## Integration with Docker Compose

The `docker-compose.yml` includes automatic dashboard provisioning:

```yaml
services:
  grafana:
    image: grafana/grafana:latest
    volumes:
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards/auth-service:ro
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
    environment:
      - GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH=/etc/grafana/provisioning/dashboards/auth-service/auth-service-overview.json
```

The default home dashboard is set to Auth Service Overview.

## Advanced Features

### Template Variables

Add variables for dynamic filtering:

```
Variable: environment
Query: label_values(environment)
Usage: {environment="$environment"}
```

### Annotations

Mark important events on graphs:

```
Example: Deployments
Query: changes(process_start_time_seconds[5m]) > 0
```

### Panel Links

Link panels to related dashboards for drill-down analysis:

```
Panel: High Error Rate
Link: Authentication Metrics Dashboard (filtered to failed logins)
```

## Performance Tips

1. Use recording rules for expensive queries
2. Limit time range to reduce query load
3. Use rate() instead of irate() for smoother graphs
4. Aggregate high-cardinality metrics
5. Cache dashboard results (Grafana Enterprise feature)

## Dashboard URLs

Direct links when running locally:

- Overview: http://localhost:3000/d/auth-service-overview
- Authentication: http://localhost:3000/d/auth-service-authentication
- Infrastructure: http://localhost:3000/d/auth-service-infrastructure

## Additional Resources

- Prometheus Queries: `docs/PROMETHEUS_QUERIES.md` - 100+ example queries
- Metrics Documentation: `internal/observability/README.md` - All available metrics
- Grafana Docs: https://grafana.com/docs/grafana/latest/
- PromQL Guide: https://prometheus.io/docs/prometheus/latest/querying/basics/

## Getting Help

If you encounter issues:

1. Check the Troubleshooting section above
2. Review Prometheus metrics at http://localhost:9090
3. Check service logs: `docker-compose logs -f auth-service`
4. Review Grafana logs: `docker-compose logs -f grafana`

## Summary

This setup includes 3 comprehensive dashboards with 34 total panels covering all service aspects. Dashboards are automatically provisioned with Docker Compose and are fully customizable with export/import support.

Next steps:

1. Start services: `docker-compose --profile monitoring up -d`
2. Open Grafana: http://localhost:3000 (admin/admin)
3. Browse dashboards under Browse → Auth Service
4. Configure alerts for critical metrics
5. Customize as needed for your environment
