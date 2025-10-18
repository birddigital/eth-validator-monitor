# Claude Code Project Configuration

This directory contains Claude Code-specific configuration for the eth-validator-monitor project.

## Specialized Agents

### `/go-crypto` - Golang & Crypto Development Agent

A specialized agent with deep expertise in:
- **Golang**: Idiomatic patterns, concurrency, testing, performance
- **Ethereum/Blockchain**: Beacon Chain, validators, consensus mechanics
- **Project Patterns**: Validator monitoring, metrics, API integration

**Usage Examples:**

```bash
/go-crypto How should I implement validator effectiveness scoring?
/go-crypto Review this goroutine for race conditions
/go-crypto Best way to structure Prometheus metrics for validators?
/go-crypto Explain Beacon Chain attestation mechanics
```

**When to Use:**
- Implementing Go code
- Working with Ethereum/validator concepts
- Performance optimization
- Testing strategies
- Prometheus metrics
- Database patterns

**Features:**
- Context-aware of project structure
- Follows Go best practices
- Understands crypto/blockchain domain
- Provides code + tests
- Security-conscious
- Performance-oriented

**NightShift Integration:**

NightShift automatically uses `/go-crypto` when:
- Implementing Go code for validators, metrics, or APIs
- Debugging Ethereum/Beacon Chain integration
- Optimizing performance-critical paths
- Designing Prometheus metrics
- Reviewing crypto-related code

NightShift will invoke: `/go-crypto <specific question>` to get specialized guidance before implementation.

## Custom Commands

### `/orchestrate`
Orchestrates all pending tasks in a project with concurrent AI agents.

### `/nightshift`
Autonomous development orchestrator with context management and sub-agent delegation.

**Key Features:**
- Auto-continues to next task after completion
- Monitors context usage (stops at 5% remaining)
- Uses sub-agents for exploration to preserve main context
- Integrates with Task Master for workflow management

## Settings

### `settings.local.json`

Project-specific permissions and configurations:

- **Task Master Integration**: Full access to task-master CLI and MCP tools
- **Autonomous Operations**: NightShift orchestrator enabled
- **Development Tools**: Go, git, npm, docker, redis-cli
- **File Operations**: Read, Edit, Write, Glob, Grep
- **Custom Commands**: Access to project-specific slash commands

### Permissions Philosophy

This project uses `--dangerously-skip-permissions` for autonomous development sessions. All allowed tools are explicitly listed for transparency.

**Auto-allowed tools:**
- Task Master operations
- Git operations
- Go development tools (gofmt, golint, staticcheck)
- File operations within project
- Prometheus/Grafana tools

## Workflow Integration

### NightShift Auto-Continue

The bye script (`~/.claude/scripts/bye`) has been configured for intelligent session chaining:

**Thresholds:**
- **Quick Session**: < 2 minutes (minimal reporting, blog post generation)
- **Blog Generation**: ≥ 2 minutes with achievements
- **Auto-Continue**: Bypasses quick mode to spawn next session

**How It Works:**

1. Session completes, calls `bye --auto-continue`
2. Script collects metrics (git, Task Master, achievements)
3. If Task Master has next task:
   - Fetches task details
   - Creates prompt with context
   - Launches new Claude session with task
4. If no tasks available:
   - Creates general continuation prompt
   - Launches session with previous context

**Session Context Preserved:**
- Duration and timestamps
- Files changed, commits made
- Task Master progress
- NightShift insights and recommendations
- Code changes and achievements

## Task Master Integration

This project uses Task Master for development workflow:

**Key Commands:**
```bash
task-master next              # Get next available task
task-master show <id>        # View task details
task-master set-status        # Update task status
task-master update-subtask    # Log implementation notes
```

**Slash Commands:**
- All `task-master` CLI commands available via MCP
- See `.taskmaster/CLAUDE.md` for complete guide

## Development Best Practices

### Context Management (NightShift)

**Critical Rules:**
1. At 10% context: Start wrapping up current subtask
2. At 5% context: MUST call `bye --auto-continue` immediately
3. After EVERY subtask: Call `bye --auto-continue`
4. Use Task tool (sub-agents) for exploration
5. Only use main context for implementation

**Why?**
- Prevents context exhaustion
- Enables continuous development
- Preserves session knowledge
- Allows parallel work

### Sub-Agent Usage

**Use sub-agents for:**
- Codebase exploration (`Explore` agent)
- Research and documentation lookup
- Code review
- Test generation
- Analysis and reporting

**Example:**
```javascript
Task(
  subagent_type: "Explore",
  prompt: "Find all Prometheus metric definitions in codebase"
)
```

## Quick Start for New Projects

Want to set up NightShift in a new project?

```bash
# Navigate to your project
cd /path/to/your/project

# Run one command to set everything up
~/.claude/scripts/setup-nightshift

# Initialize Task Master (if not done)
task-master init
task-master parse-prd .taskmaster/docs/prd.txt

# Start autonomous development
claude --dangerously-skip-permissions
/nightshift
```

That's it! NightShift will:
1. Find the next available task with `task-master next`
2. Break it down and implement it
3. Mark it complete
4. Auto-continue to the next task
5. Repeat until all tasks are done

## Troubleshooting

### Auto-continue not working

**Issue**: bye script exits without spawning new session

**Fix**: ✅ Fixed in this session
- Script now checks `AUTO_CONTINUE` flag before quick-exit
- Quick session threshold changed to 2 minutes
- Auto-continue bypasses quick mode

**Test:**
```bash
~/.claude/scripts/bye --auto-continue
```

### NightShift not finding tasks

**Issue**: "No tasks found" or working on wrong task

**Fix**: ✅ Fixed in this session
- NightShift now always calls `task-master next` dynamically
- No hardcoded task IDs
- Works with any Task Master project

### New project setup takes too long

**Solution**: Use the setup script!
```bash
~/.claude/scripts/setup-nightshift
```
Configures everything in seconds

### Permissions denied

**Solution**: Check `.claude/settings.local.json`
- Ensure tool is in `allow` list
- Use pattern matching (e.g., `Bash(go:*)` for all go commands)
- Restart Claude Code after settings changes

### Task Master not available

**Check:**
1. Is task-master installed? `which task-master`
2. Is project initialized? `ls .taskmaster/tasks/tasks.json`
3. Is MCP server configured? Check `.mcp.json`

## Recent Updates

### 2025-10-18 - NightShift Improvements

1. **Fixed bye script auto-continue bypass**
   - Auto-continue now works correctly for all session durations
   - Script checks `AUTO_CONTINUE` flag before quick-exit
   - Enables proper session chaining

2. **Changed session thresholds**
   - Quick session: 5min → 2min
   - Blog generation: 10min → 2min
   - Rationale: Short sessions can have significant accomplishments

3. **Fixed NightShift dynamic task lookup**
   - Removed hardcoded task examples (7.4.1, etc.)
   - NightShift now always uses `task-master next` to find tasks dynamically
   - Works across any project with Task Master initialized

4. **Created `/go-crypto` specialized agent**
   - Golang expertise (concurrency, testing, performance)
   - Ethereum/blockchain domain knowledge
   - Project-aware patterns and conventions
   - NightShift automatically uses for Go/crypto work

5. **Created NightShift setup automation**
   - New setup script: `~/.claude/scripts/setup-nightshift`
   - Template: `~/.claude/templates/nightshift-settings.json`
   - Automatically configures new projects for autonomous development
   - Intelligently merges with existing settings

---

**Project**: eth-validator-monitor
**Purpose**: Monitor Ethereum validator performance with metrics and alerts
**Stack**: Go, PostgreSQL, GraphQL, Prometheus, Grafana
