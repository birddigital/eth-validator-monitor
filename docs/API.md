# Ethereum Validator Monitor - GraphQL API Documentation

## Overview

The Ethereum Validator Monitor provides a GraphQL API for querying and monitoring Ethereum validator performance, snapshots, alerts, and network statistics.

## Authentication

The API supports two authentication methods:

### API Key Authentication

Include your API key in the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ validator(index: 123) { pubkey } }"}' \
  http://localhost:8080/graphql
```

### Bearer Token Authentication

Include a JWT token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ validator(index: 123) { pubkey } }"}' \
  http://localhost:8080/graphql
```

## Rate Limiting

The API implements rate limiting to prevent abuse:

- Default: 100 requests per second per API key
- Burst capacity: 200 requests
- Exceeding limits returns HTTP 429 (Too Many Requests)

## Schema Overview

### Core Types

#### Validator
```graphql
type Validator {
  validatorIndex: Int!
  pubkey: String!
  status: ValidatorStatus!
  effectiveBalance: BigInt
  latestSnapshot: ValidatorSnapshot
  snapshots(filter: SnapshotFilterInput, pagination: PaginationInput): SnapshotConnection!
  alerts(filter: AlertFilterInput, pagination: PaginationInput): AlertConnection!
  performanceMetrics(epochFrom: Int, epochTo: Int): [PerformanceMetrics!]!
}
```

#### ValidatorSnapshot
```graphql
type ValidatorSnapshot {
  time: Time!
  validatorIndex: Int!
  epoch: Int!
  slot: Int!
  balance: BigInt!
  effectiveBalance: BigInt!
  attestationEffectiveness: Float!
  attestationInclusionDelay: Int!
  attestationHeadVote: Boolean!
  attestationSourceVote: Boolean!
  attestationTargetVote: Boolean!
  proposed: Boolean
  validator: Validator!
}
```

#### Alert
```graphql
type Alert {
  id: String!
  validatorIndex: Int!
  type: AlertType!
  severity: AlertSeverity!
  message: String!
  metadata: String
  resolved: Boolean!
  createdAt: Time
  resolvedAt: Time
  validator: Validator!
}
```

### Enums

```graphql
enum ValidatorStatus {
  PENDING_INITIALIZED
  PENDING_QUEUED
  ACTIVE_ONGOING
  ACTIVE_EXITING
  ACTIVE_SLASHED
  EXITED_UNSLASHED
  EXITED_SLASHED
  WITHDRAWAL_POSSIBLE
  WITHDRAWAL_DONE
  UNKNOWN
}

enum AlertType {
  MISSED_ATTESTATION
  POOR_EFFECTIVENESS
  BALANCE_DECREASED
  SLASHING_RISK
  OFFLINE
  SYNC_COMMITTEE_MISS
}

enum AlertSeverity {
  INFO
  WARNING
  CRITICAL
  EMERGENCY
}
```

## Queries

### Get Single Validator

```graphql
query GetValidator {
  validator(index: 123) {
    validatorIndex
    pubkey
    status
    effectiveBalance
    latestSnapshot {
      time
      balance
      attestationEffectiveness
    }
  }
}
```

### List Validators with Pagination

```graphql
query ListValidators {
  validators(
    filter: {
      monitored: true
      status: ACTIVE_ONGOING
    }
    pagination: {
      limit: 50
      cursor: "base64cursor"
    }
  ) {
    edges {
      node {
        validatorIndex
        pubkey
        status
      }
      cursor
    }
    pageInfo {
      hasNextPage
      hasPreviousPage
      startCursor
      endCursor
    }
    totalCount
  }
}
```

### Get Validator Snapshots

```graphql
query GetSnapshots {
  validator(index: 123) {
    snapshots(
      filter: {
        minEffectiveness: 95.0
      }
      pagination: {
        limit: 100
      }
    ) {
      edges {
        node {
          time
          balance
          attestationEffectiveness
          attestationInclusionDelay
        }
      }
      totalCount
    }
  }
}
```

### Get Active Alerts

```graphql
query GetAlerts {
  alerts(
    filter: {
      validatorIndex: 123
      severity: CRITICAL
      resolved: false
    }
    pagination: {
      limit: 20
    }
  ) {
    edges {
      node {
        id
        type
        severity
        message
        createdAt
        validator {
          validatorIndex
          pubkey
        }
      }
    }
    totalCount
  }
}
```

### Get Network Statistics

```graphql
query GetNetworkStats {
  networkStats {
    totalActiveValidators
    totalStake
    avgAttestationEffectiveness
    lastUpdated
  }
}
```

### Get Performance Metrics

```graphql
query GetPerformanceMetrics {
  validatorPerformance(
    validatorIndex: 123
    epochFrom: 1000
    epochTo: 1100
  ) {
    epoch
    attestationsCount
    correctHeadVotes
    correctSourceVotes
    correctTargetVotes
    effectivenessScore
  }
}
```

## Mutations

### Add Validator

```graphql
mutation AddValidator {
  addValidator(
    pubkey: "0x1234..."
    validatorIndex: 123
  ) {
    validatorIndex
    pubkey
    status
  }
}
```

### Remove Validator

```graphql
mutation RemoveValidator {
  removeValidator(validatorIndex: 123)
}
```

### Update Monitoring Status

```graphql
mutation UpdateMonitoring {
  updateValidatorMonitoring(
    validatorIndex: 123
    monitored: true
  ) {
    validatorIndex
    monitored
  }
}
```

### Resolve Alert

```graphql
mutation ResolveAlert {
  resolveAlert(id: "alert-123") {
    id
    resolved
    resolvedAt
  }
}
```

### Bulk Resolve Alerts

```graphql
mutation BulkResolveAlerts {
  bulkResolveAlerts(
    filter: {
      validatorIndex: 123
      severity: WARNING
    }
  )
}
```

### Trigger Data Collection

```graphql
mutation TriggerCollection {
  triggerCollection(validatorIndex: 123)
}
```

## Subscriptions

### Subscribe to New Snapshots

```graphql
subscription OnSnapshotAdded {
  validatorSnapshotAdded(validatorIndex: 123) {
    time
    balance
    attestationEffectiveness
  }
}
```

### Subscribe to Alert Creation

```graphql
subscription OnAlertCreated {
  alertCreated(
    validatorIndex: 123
    severity: CRITICAL
  ) {
    id
    type
    severity
    message
    createdAt
  }
}
```

### Subscribe to Alert Resolution

```graphql
subscription OnAlertResolved {
  alertResolved(validatorIndex: 123) {
    id
    resolvedAt
  }
}
```

### Subscribe to Network Stats

```graphql
subscription OnNetworkStatsUpdated {
  networkStatsUpdated {
    totalActiveValidators
    totalStake
    avgAttestationEffectiveness
    lastUpdated
  }
}
```

## Filtering

Most connection queries support filtering:

### Validator Filters

```graphql
input ValidatorFilterInput {
  indices: [Int!]
  pubkeys: [String!]
  status: ValidatorStatus
  monitored: Boolean
  minBalance: BigInt
  maxBalance: BigInt
}
```

### Snapshot Filters

```graphql
input SnapshotFilterInput {
  validatorIndex: Int
  startTime: Time
  endTime: Time
  minEffectiveness: Float
  maxEffectiveness: Float
}
```

### Alert Filters

```graphql
input AlertFilterInput {
  validatorIndex: Int
  type: AlertType
  severity: AlertSeverity
  resolved: Boolean
}
```

## Pagination

All connection queries use cursor-based pagination:

```graphql
input PaginationInput {
  limit: Int      # Default: 50, Max: 100
  cursor: String  # Base64-encoded cursor from previous response
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}
```

### Pagination Example

```javascript
// First page
let cursor = null;
let allValidators = [];

while (true) {
  const response = await query({
    validators(pagination: { limit: 100, cursor }) {
      edges {
        node { validatorIndex pubkey }
        cursor
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  });

  allValidators.push(...response.validators.edges);

  if (!response.validators.pageInfo.hasNextPage) {
    break;
  }

  cursor = response.validators.pageInfo.endCursor;
}
```

## Error Handling

### Standard Error Response

```json
{
  "errors": [
    {
      "message": "Validator not found",
      "path": ["validator"],
      "extensions": {
        "code": "NOT_FOUND",
        "validatorIndex": 123
      }
    }
  ]
}
```

### Common Error Codes

- `UNAUTHENTICATED`: No valid credentials provided
- `UNAUTHORIZED`: Insufficient permissions
- `NOT_FOUND`: Requested resource doesn't exist
- `INVALID_ARGUMENT`: Invalid input parameters
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `INTERNAL_ERROR`: Server error

## Performance Optimization

### Using DataLoader

The API automatically batches and caches queries using DataLoader to prevent N+1 queries:

```graphql
# This query efficiently batches validator fetches
query GetMultipleValidators {
  v1: validator(index: 1) { pubkey }
  v2: validator(index: 2) { pubkey }
  v3: validator(index: 3) { pubkey }
}
```

### Field Selection

Only request fields you need to minimize response size:

```graphql
# Good - only required fields
query {
  validators(pagination: { limit: 100 }) {
    edges {
      node {
        validatorIndex
        status
      }
    }
  }
}

# Avoid - requesting all fields
query {
  validators(pagination: { limit: 100 }) {
    edges {
      node {
        validatorIndex
        pubkey
        status
        latestSnapshot {
          # ... many fields
        }
        alerts {
          # ... many fields
        }
      }
    }
  }
}
```

### Caching

Responses include cache TTL information:

- Validator metadata: 5 minutes
- Latest snapshot: 1 minute
- Historical snapshots: 1 hour
- Network stats: 30 seconds
- Alerts: 30 seconds

## Best Practices

1. **Use pagination** for large result sets
2. **Request only needed fields** to reduce payload size
3. **Batch queries** when fetching multiple validators
4. **Use subscriptions** for real-time updates instead of polling
5. **Implement retry logic** with exponential backoff
6. **Cache responses** on the client side when appropriate
7. **Monitor rate limits** and implement queueing if needed

## Code Examples

See `/examples` directory for full code samples:

- `javascript/`: Node.js examples with Apollo Client
- `python/`: Python examples with GQL
- `go/`: Go examples with graphql-go
- `curl/`: Bash script examples

## Support

For API issues or questions:
- GitHub Issues: https://github.com/your-org/eth-validator-monitor/issues
- Documentation: https://docs.eth-validator-monitor.com
