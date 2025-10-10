#!/bin/bash

# Ethereum Validator Monitor - cURL Examples
#
# Usage:
#   export API_KEY="your-api-key"
#   ./queries.sh

API_URL="${API_URL:-http://localhost:8080/graphql}"
API_KEY="${API_KEY:-your-api-key-here}"

# Helper function to execute GraphQL query
execute_query() {
  local query="$1"
  local variables="${2:-{}}"

  echo "Executing query..."
  curl -s -X POST "$API_URL" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d "$(jq -n --arg query "$query" --argjson variables "$variables" '{query: $query, variables: $variables}')" \
    | jq '.'
}

# Example 1: Get single validator
get_validator() {
  local index="$1"

  echo "=== Get Validator $index ==="

  query='
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
        }
      }
    }
  '

  variables="{\"index\": $index}"

  execute_query "$query" "$variables"
}

# Example 2: List validators
list_validators() {
  local limit="${1:-50}"

  echo "=== List Validators (limit: $limit) ==="

  query='
    query ListValidators($limit: Int) {
      validators(pagination: {limit: $limit}) {
        edges {
          node {
            validatorIndex
            pubkey
            status
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
        totalCount
      }
    }
  '

  variables="{\"limit\": $limit}"

  execute_query "$query" "$variables"
}

# Example 3: Get validator snapshots
get_snapshots() {
  local index="$1"
  local limit="${2:-10}"

  echo "=== Get Snapshots for Validator $index ==="

  query='
    query GetSnapshots($index: Int!) {
      validator(index: $index) {
        snapshots(pagination: {limit: 10}) {
          edges {
            node {
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
      }
    }
  '

  variables="{\"index\": $index}"

  execute_query "$query" "$variables"
}

# Example 4: Get active alerts
get_alerts() {
  local index="${1:-}"

  echo "=== Get Active Alerts ==="

  if [ -n "$index" ]; then
    query='
      query GetAlerts($index: Int!) {
        alerts(
          filter: {validatorIndex: $index, resolved: false}
          pagination: {limit: 20}
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
    '
    variables="{\"index\": $index}"
  else
    query='
      query GetAllAlerts {
        alerts(
          filter: {resolved: false}
          pagination: {limit: 50}
        ) {
          edges {
            node {
              id
              validatorIndex
              type
              severity
              message
              createdAt
            }
          }
          totalCount
        }
      }
    '
    variables='{}'
  fi

  execute_query "$query" "$variables"
}

# Example 5: Get network stats
get_network_stats() {
  echo "=== Get Network Stats ==="

  query='
    query GetNetworkStats {
      networkStats {
        totalActiveValidators
        totalStake
        avgAttestationEffectiveness
        lastUpdated
      }
    }
  '

  execute_query "$query"
}

# Example 6: Mutation - Add validator
add_validator() {
  local pubkey="$1"
  local index="$2"

  echo "=== Add Validator ==="

  query='
    mutation AddValidator($pubkey: String!, $index: Int!) {
      addValidator(pubkey: $pubkey, validatorIndex: $index) {
        validatorIndex
        pubkey
        status
      }
    }
  '

  variables="{\"pubkey\": \"$pubkey\", \"index\": $index}"

  execute_query "$query" "$variables"
}

# Example 7: Mutation - Resolve alert
resolve_alert() {
  local alert_id="$1"

  echo "=== Resolve Alert ==="

  query='
    mutation ResolveAlert($id: String!) {
      resolveAlert(id: $id) {
        id
        resolved
        resolvedAt
      }
    }
  '

  variables="{\"id\": \"$alert_id\"}"

  execute_query "$query" "$variables"
}

# Main menu
show_menu() {
  echo ""
  echo "Ethereum Validator Monitor - cURL Examples"
  echo "==========================================="
  echo "1. Get single validator"
  echo "2. List validators"
  echo "3. Get validator snapshots"
  echo "4. Get active alerts"
  echo "5. Get network stats"
  echo "6. Add validator (mutation)"
  echo "7. Resolve alert (mutation)"
  echo "0. Exit"
  echo ""
}

# Main execution
if [ "$#" -eq 0 ]; then
  # Interactive mode
  while true; do
    show_menu
    read -p "Select an option: " option

    case $option in
      1)
        read -p "Enter validator index: " index
        get_validator "$index"
        ;;
      2)
        read -p "Enter limit (default 50): " limit
        limit="${limit:-50}"
        list_validators "$limit"
        ;;
      3)
        read -p "Enter validator index: " index
        get_snapshots "$index"
        ;;
      4)
        read -p "Enter validator index (leave empty for all): " index
        get_alerts "$index"
        ;;
      5)
        get_network_stats
        ;;
      6)
        read -p "Enter pubkey: " pubkey
        read -p "Enter validator index: " index
        add_validator "$pubkey" "$index"
        ;;
      7)
        read -p "Enter alert ID: " alert_id
        resolve_alert "$alert_id"
        ;;
      0)
        echo "Exiting..."
        exit 0
        ;;
      *)
        echo "Invalid option"
        ;;
    esac

    echo ""
    read -p "Press Enter to continue..."
  done
else
  # Command-line mode
  case "$1" in
    validator)
      get_validator "${2:-123}"
      ;;
    list)
      list_validators "${2:-50}"
      ;;
    snapshots)
      get_snapshots "${2:-123}"
      ;;
    alerts)
      get_alerts "$2"
      ;;
    stats)
      get_network_stats
      ;;
    *)
      echo "Usage: $0 [validator|list|snapshots|alerts|stats] [args...]"
      exit 1
      ;;
  esac
fi
