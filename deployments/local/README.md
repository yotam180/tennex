# Tennex Development Shell Shortcuts

Simple shell shortcuts to streamline Tennex development workflow.

## Quick Setup

Add to your `~/.zshrc`:

```bash
# Tennex Development Environment  
alias txsetup="source /Users/yotam/projects/tennex/deployments/local/shell_shortcuts.sh"
```

Reload: `source ~/.zshrc`

## Usage

```bash
txsetup    # Load all Tennex shortcuts
txhelp     # Show all available commands
```

## Key Commands

```bash
# Quick Development
txdev                    # Start infra + setup for local development
txinfra                  # Start infrastructure only  
txup                     # Start all services in Docker

# Local Go Development
txrb                     # Build and run backend
txres                    # Build and run eventstream  
txrbridge               # Build and run bridge

# Docker Operations
txps                     # Show service status
txl [service]           # Show logs
txdown                  # Stop services

# Utilities
txgen                   # Generate code from contracts
txtest                  # Run tests
txcode                  # Open Cursor
```

All services run on standard ports:
- Backend: `localhost:8000`
- EventStream: `localhost:6002`  
- Bridge: `localhost:6003`
- PgAdmin: `http://localhost:8080`
- NATS Monitor: `http://localhost:8222`
- MinIO Console: `http://localhost:9001`
