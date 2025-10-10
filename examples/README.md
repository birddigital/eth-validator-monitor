# Ethereum Validator Monitor - API Examples

This directory contains code examples demonstrating how to use the GraphQL API.

## Directory Structure

```
examples/
├── javascript/    # Node.js examples
├── python/        # Python examples
├── go/            # Go examples
└── curl/          # Shell script examples
```

## Prerequisites

### For JavaScript Examples

```bash
cd javascript
npm install @apollo/client graphql cross-fetch
```

### For Python Examples

```bash
cd python
pip install gql[all] python-dotenv
```

### For Go Examples

```bash
cd go
go mod init example
go get github.com/machinebox/graphql
```

### For cURL Examples

- `jq` for JSON processing: `brew install jq` (macOS) or `apt-get install jq` (Linux)

## Authentication

Set your API key as an environment variable:

```bash
export API_KEY="your-api-key-here"
```

Or create a `.env` file in each example directory:

```
API_KEY=your-api-key-here
API_URL=http://localhost:8080/graphql
```

## Running Examples

### JavaScript

```bash
cd javascript
node query-validators.js
```

### Python

```bash
cd python
python query_validators.py
```

### Go

```bash
cd go
go run query_validators.go
```

### cURL (Interactive)

```bash
cd curl
./queries.sh
```

### cURL (Command-line)

```bash
cd curl
./queries.sh validator 123
./queries.sh list 50
./queries.sh snapshots 123
./queries.sh alerts
./queries.sh stats
```

## Examples Included

### 1. Query Single Validator

Fetch detailed information about a specific validator:

- Validator index, pubkey, status
- Effective balance
- Latest snapshot data
- Attestation effectiveness

### 2. List Validators with Pagination

Retrieve all validators with cursor-based pagination:

- Filter by monitoring status
- Paginate through large datasets
- Handle pagination cursors

### 3. Get Validator Snapshots

Fetch historical snapshot data:

- Time-series balance data
- Attestation performance metrics
- Inclusion delays and votes

### 4. Get Active Alerts

Query alerts for validators:

- Filter by severity
- Filter by resolution status
- Validator-specific alerts

### 5. Get Network Statistics

Retrieve aggregate network metrics:

- Total active validators
- Total stake
- Average attestation effectiveness

### 6. Add Validator (Mutation)

Add a new validator to the monitoring system:

- Register by pubkey and index
- Set monitoring status

### 7. Resolve Alert (Mutation)

Mark alerts as resolved:

- Single alert resolution
- Bulk alert resolution

## Common Patterns

### Error Handling

```javascript
try {
  const { data } = await client.query({ query, variables });
  console.log(data);
} catch (error) {
  if (error.graphQLErrors) {
    error.graphQLErrors.forEach(({ message, extensions }) => {
      console.error(`GraphQL error: ${message}`, extensions);
    });
  }
  if (error.networkError) {
    console.error('Network error:', error.networkError);
  }
}
```

### Pagination Loop

```javascript
let cursor = null;
let allItems = [];

while (true) {
  const { data } = await query({ pagination: { limit: 100, cursor } });
  allItems = allItems.concat(data.edges.map(e => e.node));

  if (!data.pageInfo.hasNextPage) break;
  cursor = data.pageInfo.endCursor;
}
```

### Rate Limit Handling

```javascript
async function queryWithRetry(queryFn, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await queryFn();
    } catch (error) {
      if (error.extensions?.code === 'RATE_LIMIT_EXCEEDED') {
        const delay = Math.pow(2, i) * 1000; // Exponential backoff
        await new Promise(resolve => setTimeout(resolve, delay));
        continue;
      }
      throw error;
    }
  }
  throw new Error('Max retries exceeded');
}
```

## Advanced Examples

For more advanced examples, see:

- **Subscriptions**: Real-time updates for alerts and snapshots
- **Batch Operations**: Efficiently query multiple validators
- **Custom Filters**: Complex filtering and sorting
- **Performance Optimization**: Caching and field selection

## Testing

Each example includes basic error handling and logging. For production use:

1. Implement proper retry logic with exponential backoff
2. Add comprehensive error handling
3. Implement response caching
4. Monitor API rate limits
5. Log all API interactions for debugging

## Support

For questions or issues with these examples:

- Check the main [API Documentation](../docs/API.md)
- Open an issue on GitHub
- Consult the GraphQL schema at `http://localhost:8080/graphql`

## License

These examples are provided as-is for demonstration purposes.
