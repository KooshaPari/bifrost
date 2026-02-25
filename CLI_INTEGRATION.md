# Bifrost CLI Integration Guide

## Overview

The Bifrost CLI can be integrated with desktop applications (.app for macOS, .exe for Windows) to provide a unified interface for both infrastructure management and user-facing operations.

## Desktop Application Integration

### macOS Integration (.app)

#### Embedding CLI in App Bundle
```bash
# Copy CLI binary to app bundle
cp bin/bifrost MyApp.app/Contents/MacOS/bifrost

# Make executable
chmod +x MyApp.app/Contents/MacOS/bifrost
```

#### IPC Communication
Use Unix sockets for communication between GUI and CLI:

```swift
// Swift example
import Foundation

class BifrostCLI {
    func runCommand(_ command: String, args: [String]) -> String {
        let process = Process()
        process.executableURL = Bundle.main.url(forResource: "bifrost", withExtension: nil)
        process.arguments = [command] + args
        
        let pipe = Pipe()
        process.standardOutput = pipe
        try? process.run()
        process.waitUntilExit()
        
        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        return String(data: data, encoding: .utf8) ?? ""
    }
}
```

### Windows Integration (.exe)

#### Embedding CLI in App
```csharp
// C# example
using System.Diagnostics;

class BifrostCLI {
    public string RunCommand(string command, params string[] args) {
        var process = new ProcessStartInfo {
            FileName = "bifrost.exe",
            Arguments = $"{command} {string.Join(" ", args)}",
            UseShellExecute = false,
            RedirectStandardOutput = true,
            CreateNoWindow = true
        };
        
        using (var proc = Process.Start(process)) {
            return proc.StandardOutput.ReadToEnd();
        }
    }
}
```

## GUI Integration Patterns

### Pattern 1: Menu-Based Commands
```
File Menu
├── Initialize Project
├── Open Configuration
└── Deploy
    ├── Deploy to Fly.io
    ├── Deploy to Vercel
    └── Deploy to Railway

Tools Menu
├── Start Server
├── Stop Server
├── View Logs
└── Configuration
    ├── Show Config
    ├── Validate Config
    └── Set API Keys

Plugins Menu
├── List Plugins
├── Enable Plugin
└── Disable Plugin
```

### Pattern 2: Status Bar Integration
```
Status Bar
├── Server Status (Running/Stopped)
├── Active Plugins (3/8)
├── Deployment Status
└── Quick Actions
    ├── Start Server
    ├── Deploy
    └── Settings
```

### Pattern 3: Sidebar Navigation
```
Sidebar
├── Dashboard
├── Server
│   ├── Start/Stop
│   ├── Logs
│   └── Configuration
├── Deployment
│   ├── Fly.io
│   ├── Vercel
│   ├── Railway
│   └── Render
├── Plugins
│   ├── List
│   ├── Enable/Disable
│   └── Configuration
├── Datasets
│   ├── List
│   ├── Load
│   └── Statistics
└── Settings
```

## API Endpoints for GUI

### Server Management
```bash
bifrost server --port 8080 --host 0.0.0.0
bifrost server --help
```

### Deployment
```bash
bifrost deploy fly --dry-run
bifrost deploy vercel
bifrost deploy railway
```

### Configuration
```bash
bifrost config show
bifrost config set KEY VALUE
bifrost config validate
```

### Plugins
```bash
bifrost plugin list
bifrost plugin enable promptadapter
bifrost plugin disable embeddings
```

### Datasets
```bash
bifrost dataset list
bifrost dataset load cursor
bifrost dataset stats
```

## Real-Time Monitoring

### Log Streaming
```bash
# Start server with verbose output
bifrost server -v -l debug

# Capture output in GUI
```

### Status Polling
```bash
# Poll server status
bifrost server --status

# Check deployment status
bifrost deploy --status
```

## Configuration Management

### Shared Configuration
```
~/.bifrost/
├── config/
│   ├── bifrost.yaml
│   ├── plugins.yaml
│   └── deployment.yaml
├── data/
│   ├── datasets/
│   └── models/
└── logs/
    ├── server.log
    └── deployment.log
```

### Environment Variables
```bash
# Set in GUI, persist in .env
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
BIFROST_PORT=8080
BIFROST_HOST=0.0.0.0
```

## Error Handling

### CLI Error Codes
```
0   - Success
1   - General error
2   - Configuration error
3   - Deployment error
4   - Plugin error
5   - Dataset error
```

### GUI Error Display
```
Error Dialog
├── Error Code
├── Error Message
├── Suggested Action
└── View Logs Button
```

## Performance Considerations

### Async Execution
- Run long-running commands (deploy, server) in background threads
- Show progress indicators
- Allow cancellation

### Caching
- Cache plugin list
- Cache dataset list
- Cache configuration

### Resource Management
- Limit concurrent CLI processes
- Clean up temporary files
- Monitor memory usage

## Testing Integration

### Unit Tests
```bash
go test ./cmd/bifrost/cli/...
```

### Integration Tests
```bash
# Test CLI with GUI
./bin/bifrost init
./bin/bifrost config validate
./bin/bifrost plugin list
```

### End-to-End Tests
```bash
# Full workflow
./bin/bifrost init
./bin/bifrost config set OPENAI_API_KEY sk-...
./bin/bifrost config validate
./bin/bifrost server --dry-run
./bin/bifrost deploy fly --dry-run
```

## Distribution

### macOS
```bash
# Create .app bundle
mkdir -p MyApp.app/Contents/MacOS
cp bin/bifrost MyApp.app/Contents/MacOS/
chmod +x MyApp.app/Contents/MacOS/bifrost

# Code sign
codesign -s - MyApp.app
```

### Windows
```bash
# Copy executable
copy bin\bifrost.exe MyApp\bifrost.exe

# Create installer
# Use NSIS or WiX toolset
```

## Next Steps

1. Integrate CLI with macOS app
2. Integrate CLI with Windows app
3. Create GUI wrappers for common commands
4. Add real-time monitoring
5. Implement configuration UI
6. Add deployment wizard

