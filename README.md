# eth-validator-monitor

A production-quality Go application for monitoring Ethereum validators with real-time performance tracking, alerts, and comprehensive analytics.

## Features

- 🔍 **Real-time Monitoring**: Track validator status, attestations, and proposals
- 📊 **Performance Analytics**: Score validators and compare against network averages
- 🚨 **Smart Alerting**: Get notified of critical events and performance degradation
- 📈 **Historical Data**: Track trends and analyze performance over time
- 🎯 **GraphQL API**: Flexible querying with subscriptions for real-time updates
- 🐳 **Easy Deployment**: Docker Compose setup for quick starts
- 📉 **Observability**: Prometheus metrics and Grafana dashboards included

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Ethereum Beacon Node RPC access (Infura, Alchemy, or local node)

### Installation

```bash
# Clone the repository
git clone https://github.com/birddigital/eth-validator-monitor
cd eth-validator-monitor

# Copy environment template
cp .env.example .env
# Edit .env with your RPC URLs and database credentials

# Start infrastructure services
docker-compose up -d postgres redis

# Run database migrations
make migrate-up

# Build the application
make build

# Run the server
make run
```

The API will be available at `http://localhost:8080`

GraphQL Playground: `http://localhost:8080/graphql`

### Using Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     eth-validator-monitor                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Collector  │───▶│   Analyzer   │───▶│     API      │  │
│  │  (goroutines)│    │   (metrics)  │    │  (GraphQL)   │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                     │                    │          │
│         ▼                     ▼                    ▼          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │    Redis     │    │  PostgreSQL  │    │  Prometheus  │  │
│  │   (cache)    │    │  (storage)   │    │  (metrics)   │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Usage

### Adding Validators

Via CLI:
```bash
./bin/cli add --pubkey 0x1234...abcd --name "My Validator"
./bin/cli add --index 123456
```

Via API:
```bash
curl -X POST http://localhost:8080/api/validators \
  -H "Content-Type: application/json" \
  -d '{"pubkey": "0x1234...abcd", "name": "My Validator"}'
```

### Querying with GraphQL

```graphql
query GetValidator {
  validator(index: 123456) {
    index
    pubkey
    status
    balance {
      current
      effective
    }
    performance {
      uptimePercentage
      attestationScore
      networkPercentile
    }
  }
}
```

### Real-time Subscriptions

```graphql
subscription MonitorValidators {
  validatorUpdates(indices: [123456]) {
    index
    status
    balance { current }
    performance { uptimePercentage }
  }
}
```

## Development

### Development Approach

This project leverages modern AI-assisted development tools to maximize development velocity while maintaining high code quality. The workflow includes automated task management, intelligent code suggestions, and comprehensive testing frameworks.

### Running Tests

```bash
# Run all tests
make test

# Run integration tests
make test-integration

# Run E2E tests
make test-e2e
```

### Code Quality

```bash
# Format code
make fmt

# Run linters
make lint

# Generate code (GraphQL, mocks)
make generate
```

### Database Migrations

```bash
# Create a new migration
make migrate-create NAME=add_validators_table

# Run migrations
make migrate-up

# Rollback migrations
make migrate-down
```

### Performance Benchmarking

The project includes a comprehensive benchmarking suite to ensure system performance under load and track regressions over time.

#### Quick Start

```bash
# Install benchmarking tools
make install-tools

# Run all benchmarks
make benchmark

# Run quick benchmarks (1s duration)
make benchmark-quick
```

#### Benchmark Components

The benchmark suite covers all performance-critical paths:

1. **Database Operations** (`database_bench_test.go`)
   - Batch insert performance (100-10,000 snapshots)
   - Complex query performance with realistic filters
   - Connection pool efficiency under concurrent load
   - TimescaleDB hypertable optimizations

2. **Redis Caching** (`redis_bench_test.go`)
   - Get/set operations with serialization
   - Batch operations using pipelines
   - Cache invalidation patterns
   - TTL management

3. **Beacon Client** (`beacon_client_bench_test.go`)
   - Retry logic under various failure rates
   - Concurrent request handling
   - Latency simulation (10ms-200ms)
   - Memory usage patterns

#### Advanced Benchmarking

```bash
# Set baseline for comparisons
make benchmark-baseline

# Compare with baseline
make benchmark-compare

# Memory profiling
make benchmark-mem
make benchmark-view-mem

# CPU profiling
make benchmark-cpu
make benchmark-view-cpu

# CI benchmarks (5 iterations)
make benchmark-ci
```

#### Performance Targets

| Component | Target | Validator Count |
|-----------|--------|-----------------|
| Validator Collection | < 5s | 10,000 |
| Database Batch Insert | < 2s | 10,000 snapshots |
| GraphQL Query (p95) | < 100ms | 100 results |
| Redis Cache Get | < 1ms | Single operation |
| Beacon API Call (p95) | < 500ms | With retry logic |

#### Interpreting Results

Benchmark output format:
```
BenchmarkBatchInserts/snapshots_10000-8    50   23412340 ns/op   4280.0 inserts/sec   1245630 B/op   12340 allocs/op
```

- `50` - Number of iterations
- `23412340 ns/op` - Nanoseconds per operation
- `4280.0 inserts/sec` - Custom throughput metric
- `1245630 B/op` - Bytes allocated per operation
- `12340 allocs/op` - Number of allocations per operation

#### Regression Detection

Use `benchstat` to detect performance regressions:

```bash
# Compare two benchmark runs
benchstat benchmarks/results/baseline.txt benchmarks/results/latest.txt
```

Output shows statistical significance:
```
name                          old time/op    new time/op    delta
BenchmarkBatchInserts/10000   2.34ms ± 2%    2.12ms ± 3%   -9.40%  (p=0.000)
```

#### Continuous Benchmarking

Benchmarks run in CI on every pull request to catch performance regressions early. Results are compared against the baseline and fail if performance degrades by more than 10%.

## Configuration

Configuration is managed via environment variables and `config.yaml`:

```yaml
ethereum:
  beacon_node: ${BEACON_NODE_URL}
  execution_node: ${EXECUTION_NODE_URL}
  network: sepolia

database:
  host: ${DB_HOST}
  port: 5432
  name: validator_monitor

redis:
  host: ${REDIS_HOST}
  port: 6379

api:
  port: 8080
  rate_limit: 100
```

See `.env.example` for all available configuration options.

## Monitoring

### Prometheus Metrics

Comprehensive metrics are exposed at `http://localhost:9090/metrics`:

**Validator Performance Metrics:**
- `validator_effectiveness_score` - Validator effectiveness score (0-100)
- `validator_attestation_participation_rate` - Attestation participation rate (0-1)
- `validator_proposal_success_rate` - Block proposal success rate (0-1)
- `validator_balance_wei` - Current validator balance in Wei
- `validator_missed_attestations_total` - Total missed attestations per validator
- `validator_snapshot_lag_seconds` - Time lag between current time and last snapshot

**API & System Metrics:**
- `api_request_duration_seconds` - API request latency histogram
- `api_requests_total` - Total API requests by endpoint and status
- `api_request_errors_total` - Total API errors by type
- `db_query_duration_seconds` - Database query execution time histogram
- `db_connections_active` - Active database connections
- `system_goroutines_count` - Running goroutines
- `system_memory_alloc_bytes` - Allocated memory in bytes

**Cache Metrics:**
- `cache_hit_rate` - Cache hit rate percentage
- `cache_hits_total` - Total cache hits by type
- `cache_misses_total` - Total cache misses by type

See full metrics list at `internal/metrics/` for complete metric definitions.

### Grafana Dashboard

A comprehensive Grafana dashboard is automatically provisioned when using Docker Compose.

**Access:** `http://localhost:3000` (default credentials: `admin/admin`)

**Dashboard Structure:**

1. **Validator Health Overview**
   - Overall validator effectiveness gauge (with color thresholds)
   - Active validator count
   - Attestation success rate over 24h

2. **Validator Performance Details**
   - Performance table by validator (filterable)
   - Block proposal success rate trends
   - Balance tracking over time (in ETH)

3. **System Health & API Performance**
   - API latency percentiles (p50, p95, p99)
   - API error rate monitoring
   - Database query performance
   - Connection pool status
   - Goroutine health tracking
   - Memory usage trends
   - Cache performance metrics

4. **Alerts & Recent Issues**
   - Recent missed attestations (last 1h)
   - Total rewards and penalties

**Features:**
- Template variable for filtering by validator index
- Auto-refresh every 30 seconds
- Multiple time range options (6h default)
- Color-coded thresholds (green: healthy, yellow: warning, red: critical)

**Dashboard Files:**
- Dashboard JSON: `docker/grafana/dashboards/validator-monitoring.json`
- Provisioning config: `docker/grafana/provisioning/dashboards/default.yml`
- Datasource config: `docker/grafana/provisioning/datasources/prometheus.yml`

**Exporting/Importing Dashboards:**

To export the dashboard:
1. Navigate to the dashboard in Grafana
2. Click the share icon → Export → Save to file
3. The JSON is already version-controlled at `docker/grafana/dashboards/validator-monitoring.json`

To import a dashboard:
1. Copy JSON file to `docker/grafana/dashboards/`
2. Restart Grafana: `docker-compose restart grafana`
3. Dashboard will be auto-provisioned

### Alerting Rules

Comprehensive alerting rules are configured in `docker/prometheus/alerts.yml` covering:

#### 1. Validator Health Alerts

**ValidatorEffectivenessLow** (Warning)
- Triggers when effectiveness score < 95%
- Duration: 5 minutes
- Action: Investigate validator performance and sync status

**ValidatorEffectivenessCritical** (Critical)
- Triggers when effectiveness score < 90%
- Duration: 2 minutes
- Action: Immediate investigation required

**LowAttestationParticipation** (Warning)
- Triggers when participation rate < 98%
- Duration: 10 minutes
- Action: Check beacon node connectivity

**MissedAttestations** (Warning)
- Triggers on any missed attestations in 5 minutes
- Duration: 1 minute
- Action: Review validator client logs

**ValidatorBalanceDecreasing** (Warning)
- Triggers when balance trend is negative
- Duration: 30 minutes
- Action: Check for slashing or excessive penalties

#### 2. Beacon API Health Alerts

**BeaconAPIHighLatency** (Warning)
- Triggers when p95 latency > 2 seconds
- Duration: 5 minutes
- Action: Check beacon node load and network

**BeaconAPICriticalLatency** (Critical)
- Triggers when p95 latency > 5 seconds
- Duration: 2 minutes
- Action: Immediate investigation, consider failover

**BeaconAPIHighErrorRate** (Warning)
- Triggers when error rate > 5%
- Duration: 5 minutes
- Action: Check beacon node status and logs

**BeaconAPIDown** (Critical)
- Triggers when API is unreachable
- Duration: 1 minute
- Action: Restart service or investigate crash

#### 3. Data Freshness Alerts

**ExtendedSnapshotLag** (Warning)
- Triggers when lag > 300 seconds (~2.5 epochs)
- Duration: 5 minutes
- Action: Check data collection service

**CriticalSnapshotLag** (Critical)
- Triggers when lag > 600 seconds (~5 epochs)
- Duration: 2 minutes
- Action: Immediate investigation, may indicate stale data

**ValidatorSnapshotCollectionFailing** (Critical)
- Triggers when no snapshots collected in 10 minutes
- Duration: 5 minutes
- Action: Restart collector or check beacon node

#### 4. Database Health Alerts

**DatabaseConnectionPoolExhausted** (Warning)
- Triggers when connection pool > 90% full
- Duration: 5 minutes
- Action: Increase pool size or investigate connection leaks

**DatabaseSlowQueries** (Warning)
- Triggers when p95 query time > 1 second
- Duration: 5 minutes
- Action: Review query performance and indexes

**DatabaseWriteFailures** (Critical)
- Triggers on any write errors
- Duration: 2 minutes
- Action: Check database connectivity and disk space

#### 5. System Resource Alerts

**HighMemoryUsage** (Warning)
- Triggers when memory > 2GB
- Duration: 10 minutes
- Action: Monitor for memory leaks

**CriticalMemoryUsage** (Critical)
- Triggers when memory > 4GB
- Duration: 5 minutes
- Action: Investigate memory leak or restart service

**GoroutineLeakSuspected** (Warning)
- Triggers when goroutines > 1000
- Duration: 15 minutes
- Action: Check for goroutine leaks in code

**HighCPUUsage** (Warning)
- Triggers when CPU > 80%
- Duration: 10 minutes
- Action: Profile application for hot paths

#### 6. Cache Performance Alerts

**LowCacheHitRate** (Warning)
- Triggers when hit rate < 70%
- Duration: 10 minutes
- Action: Review cache strategy and TTL settings

**Alert Management:**

View active alerts in Prometheus:
```bash
# View alerts in browser
open http://localhost:9090/alerts

# Check alert rules are loaded
curl http://localhost:9090/api/v1/rules | jq
```

Reload configuration without restart:
```bash
# Hot-reload Prometheus config
curl -X POST http://localhost:9090/-/reload
```

**Integrating with Alertmanager (Optional):**

To send alerts to Slack, PagerDuty, email, etc:

1. Add Alertmanager to `docker-compose.yml`:
```yaml
  alertmanager:
    image: prom/alertmanager:latest
    container_name: validator-monitor-alertmanager
    ports:
      - "9093:9093"
    volumes:
      - ./docker/prometheus/alertmanager.yml:/etc/alertmanager/alertmanager.yml
    networks:
      - validator-monitor
```

2. Uncomment alertmanager section in `docker/prometheus/prometheus.yml`

3. Create `docker/prometheus/alertmanager.yml` with your notification config

4. Restart services: `docker-compose up -d`

Example Alertmanager config for Slack:
```yaml
route:
  receiver: 'slack-notifications'
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: 'slack-notifications'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#validator-alerts'
        title: 'Validator Monitor Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}: {{ .Annotations.description }}{{ end }}'
```

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Author

Built by [Adrien Bird](https://github.com/birddigital) to demonstrate Go proficiency and blockchain infrastructure understanding for the ether.fi engineering team.

## Project Status

🚧 **Work in Progress** - This project is actively being developed.

Current milestone: Foundation setup and core data collection
