# Bifrost CLI Guide

## Overview

The Bifrost CLI is a comprehensive command-line interface for managing, configuring, and deploying the Bifrost LLM gateway.

## Installation

### Build from Source
```bash
cd bifrost-extensions
go build -o bifrost ./cmd/bifrost
```

### Install Globally
```bash
go install ./cmd/bifrost
```

## Quick Start

### Initialize a Project
```bash
bifrost init
```

### Start the Server
```bash
bifrost server
```

### Deploy to Fly.io
```bash
bifrost deploy fly
```

## Commands

### `bifrost server`
Start the Bifrost LLM gateway server.

**Options:**
- `-p, --port` - Server port (default: 8080)
- `-h, --host` - Server host (default: 0.0.0.0)
- `-P, --plugins` - Plugins to load (default: router, learning, fallback)
- `-l, --log-level` - Log level (default: info)

**Example:**
```bash
bifrost server -p 8090 -P router,learning,fallback
```

### `bifrost deploy`
Deploy Bifrost to various platforms.

**Subcommands:**
- `fly` - Deploy to Fly.io
- `vercel` - Deploy to Vercel
- `railway` - Deploy to Railway
- `render` - Deploy to Render
- `homebox` - Deploy to Homebox (self-hosted)

**Options:**
- `-d, --dry-run` - Show what would be deployed

**Example:**
```bash
bifrost deploy fly
bifrost deploy vercel --dry-run
```

### `bifrost config`
Manage Bifrost configuration.

**Subcommands:**
- `show` - Show current configuration
- `set <key> <value>` - Set a configuration value
- `validate` - Validate configuration

**Example:**
```bash
bifrost config show
bifrost config set OPENAI_API_KEY sk-...
bifrost config validate
```

### `bifrost plugin`
Manage Bifrost plugins.

**Subcommands:**
- `list` - List available plugins
- `enable <plugin>` - Enable a plugin
- `disable <plugin>` - Disable a plugin
- `config <plugin>` - Show plugin configuration

**Example:**
```bash
bifrost plugin list
bifrost plugin enable promptadapter
```

### `bifrost dataset`
Manage training datasets.

**Subcommands:**
- `list` - List available datasets
- `load <dataset>` - Load a dataset
- `stats` - Show dataset statistics

**Example:**
```bash
bifrost dataset list
bifrost dataset load cursor
bifrost dataset stats
```

### `bifrost version`
Show version information.

### `bifrost init`
Initialize a new Bifrost project.

## Global Options

- `-v, --verbose` - Enable verbose output
- `-c, --config` - Config file path

## Examples

### Complete Setup
```bash
# Initialize project
bifrost init

# Configure API keys
bifrost config set OPENAI_API_KEY sk-...
bifrost config set ANTHROPIC_API_KEY sk-ant-...

# Validate configuration
bifrost config validate

# Start server
bifrost server

# In another terminal, deploy
bifrost deploy fly
```

### Development
```bash
# Start with verbose logging
bifrost server -v -l debug

# Load specific plugins
bifrost server -P router,learning

# Custom port
bifrost server -p 9000
```

### Deployment
```bash
# Dry run deployment
bifrost deploy fly --dry-run

# Deploy to Fly.io
bifrost deploy fly

# Deploy to Vercel
bifrost deploy vercel

# Deploy to Homebox
bifrost deploy homebox
```

## Configuration Files

### .env
Environment variables for API keys and configuration.

### .bifrost/config/bifrost.yaml
Main Bifrost configuration file.

### .bifrost/plugins/
Plugin-specific configurations.

## Troubleshooting

### Command not found
Make sure Bifrost is installed:
```bash
go install ./cmd/bifrost
```

### Configuration errors
Validate your configuration:
```bash
bifrost config validate
```

### Deployment issues
Use dry-run to debug:
```bash
bifrost deploy fly --dry-run
```

## Next Steps

- Read [SERVERLESS_DEPLOYMENT.md](SERVERLESS_DEPLOYMENT.md) for deployment details
- Check [DEPLOY_QUICK_START.md](DEPLOY_QUICK_START.md) for quick start guides
- Review [DEPLOYMENT_COMPARISON.md](DEPLOYMENT_COMPARISON.md) for platform comparison

