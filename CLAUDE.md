# Claude Code Instructions

## Project: Ethereum Validator Monitor

This is a **Go-based Ethereum validator monitoring system** with Prometheus metrics, GraphQL API, and PostgreSQL storage.

## Specialized Agents

### `/go-crypto` - Golang & Crypto Expert

**CRITICAL**: When working on Go code or Ethereum/validator-related features, **ALWAYS** consult the `/go-crypto` agent first.

Use `/go-crypto` for:
- Implementing Go code (validators, metrics, APIs, data collection)
- Ethereum/Beacon Chain integration and concepts
- Performance optimization and concurrency patterns
- Prometheus metrics design and implementation
- Testing strategies for blockchain data
- Database schema and query optimization
- Security considerations for crypto operations

**Example Usage:**
```
/go-crypto What's the best way to track validator effectiveness scores?
/go-crypto Review this goroutine for race conditions in the collector service
/go-crypto How should I structure Prometheus metrics for attestation rates?
```

**NightShift Autonomous Mode:**

When NightShift is working autonomously, it MUST:
1. Invoke `/go-crypto` before implementing Go code or crypto features
2. Use the specialized agent's recommendations for architecture decisions
3. Apply Go best practices and Ethereum domain knowledge
4. Follow security patterns for validator monitoring

## Task Master AI Instructions
**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
