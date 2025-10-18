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

**Autonomous Implementation Protocol:**

When working on Go code or Ethereum features (both NightShift and interactive sessions):

1. **Consultation Phase:**
   - Invoke `/go-crypto` before implementing Go code or crypto features
   - Wait for specialized agent's response with recommendations

2. **Automatic Implementation Phase:**
   - **IMMEDIATELY** implement the `/go-crypto` agent's recommendations
   - **DO NOT** pause or wait for user confirmation after receiving `/go-crypto` response
   - Follow the recommended architecture, patterns, and code structure exactly
   - Apply all Go best practices and Ethereum domain knowledge from the response
   - Implement security patterns and performance optimizations as specified

3. **Implementation Requirements:**
   - Use the specialized agent's code examples as templates
   - Follow the file structure and naming conventions recommended
   - Implement all suggested benchmarks, tests, and helpers
   - Apply memory optimization tips and performance targets
   - Add Makefile targets and CI/CD integration as recommended

## Task Master AI Instructions
**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
