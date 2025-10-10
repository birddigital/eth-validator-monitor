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

Prometheus metrics are exposed at `/metrics`:

- `validator_monitor_validators_total` - Number of monitored validators
- `validator_monitor_api_request_duration_seconds` - API latency
- `validator_monitor_rpc_calls_total` - RPC calls made
- `validator_monitor_cache_hit_rate` - Cache hit ratio

Grafana dashboards are available in `docker/grafana/dashboards/`

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Author

Built by [Adrien Bird](https://github.com/birddigital) to demonstrate Go proficiency and blockchain infrastructure understanding for the ether.fi engineering team.

## Project Status

🚧 **Work in Progress** - This project is actively being developed.

Current milestone: Foundation setup and core data collection
