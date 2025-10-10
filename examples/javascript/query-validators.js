#!/usr/bin/env node

/**
 * Example: Query validators using GraphQL API
 *
 * Install dependencies:
 *   npm install @apollo/client graphql cross-fetch
 *
 * Usage:
 *   node query-validators.js
 */

const { ApolloClient, InMemoryCache, gql, HttpLink } = require('@apollo/client');
const fetch = require('cross-fetch');

// Initialize Apollo Client
const client = new ApolloClient({
  link: new HttpLink({
    uri: 'http://localhost:8080/graphql',
    fetch,
    headers: {
      'X-API-Key': process.env.API_KEY || 'your-api-key-here',
    },
  }),
  cache: new InMemoryCache(),
});

// Query to get validator information
const GET_VALIDATOR = gql`
  query GetValidator($index: Int!) {
    validator(index: $index) {
      validatorIndex
      pubkey
      status
      effectiveBalance
      latestSnapshot {
        time
        balance
        attestationEffectiveness
        attestationInclusionDelay
        attestationHeadVote
        attestationSourceVote
        attestationTargetVote
      }
    }
  }
`;

// Query to list all validators with pagination
const LIST_VALIDATORS = gql`
  query ListValidators($limit: Int, $cursor: String, $monitored: Boolean) {
    validators(
      filter: { monitored: $monitored }
      pagination: { limit: $limit, cursor: $cursor }
    ) {
      edges {
        node {
          validatorIndex
          pubkey
          status
          latestSnapshot {
            attestationEffectiveness
          }
        }
        cursor
      }
      pageInfo {
        hasNextPage
        endCursor
      }
      totalCount
    }
  }
`;

// Get alerts for a validator
const GET_ALERTS = gql`
  query GetAlerts($validatorIndex: Int!, $severity: AlertSeverity) {
    alerts(
      filter: { validatorIndex: $validatorIndex, severity: $severity, resolved: false }
      pagination: { limit: 20 }
    ) {
      edges {
        node {
          id
          type
          severity
          message
          createdAt
        }
      }
      totalCount
    }
  }
`;

async function getValidator(index) {
  try {
    const { data } = await client.query({
      query: GET_VALIDATOR,
      variables: { index },
    });

    console.log('Validator:', JSON.stringify(data.validator, null, 2));
    return data.validator;
  } catch (error) {
    console.error('Error fetching validator:', error);
    throw error;
  }
}

async function listAllValidators(monitored = true) {
  let allValidators = [];
  let cursor = null;
  let hasNextPage = true;

  console.log('Fetching all validators...');

  while (hasNextPage) {
    try {
      const { data } = await client.query({
        query: LIST_VALIDATORS,
        variables: {
          limit: 100,
          cursor,
          monitored,
        },
      });

      const validators = data.validators.edges.map((edge) => edge.node);
      allValidators = allValidators.concat(validators);

      console.log(`Fetched ${validators.length} validators (total: ${allValidators.length})`);

      hasNextPage = data.validators.pageInfo.hasNextPage;
      cursor = data.validators.pageInfo.endCursor;
    } catch (error) {
      console.error('Error fetching validators:', error);
      throw error;
    }
  }

  console.log(`\nTotal validators fetched: ${allValidators.length}`);
  return allValidators;
}

async function getValidatorAlerts(validatorIndex, severity) {
  try {
    const { data } = await client.query({
      query: GET_ALERTS,
      variables: { validatorIndex, severity },
    });

    console.log('Alerts:', JSON.stringify(data.alerts, null, 2));
    return data.alerts.edges.map((edge) => edge.node);
  } catch (error) {
    console.error('Error fetching alerts:', error);
    throw error;
  }
}

// Main execution
async function main() {
  console.log('Ethereum Validator Monitor - JavaScript Example\n');

  // Example 1: Get single validator
  console.log('Example 1: Get single validator');
  await getValidator(123);

  // Example 2: List all validators
  console.log('\nExample 2: List all monitored validators');
  const validators = await listAllValidators(true);

  // Example 3: Get alerts for a validator
  if (validators.length > 0) {
    const firstValidator = validators[0];
    console.log(`\nExample 3: Get alerts for validator ${firstValidator.validatorIndex}`);
    await getValidatorAlerts(firstValidator.validatorIndex, 'CRITICAL');
  }

  console.log('\nDone!');
}

// Run examples
main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});
