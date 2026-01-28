# dnsdist-worker

A Go-based worker application for managing configuration updates on multiple dnsdist servers using the dnsdist control-socket protocol.

## Features

- **Multi-server Management**: Connect to and manage multiple dnsdist servers simultaneously
- **Control Socket Protocol**: Uses the dnsdist control-socket protocol for secure communication
- **HMAC Authentication**: Supports HMAC-SHA256 authentication with API keys
- **Concurrent Execution**: Execute commands on all servers in parallel
- **Configuration File**: JSON-based configuration for easy server management
- **CLI Interface**: Command-line tool for executing dnsdist commands

## Installation

```bash
go get github.com/vitistack/dnsdist-worker
```

Or build from source:

```bash
git clone https://github.com/vitistack/dnsdist-worker.git
cd dnsdist-worker
go build -o dnsdist-worker ./cmd/dnsdist-worker
```

## Configuration

Create a `config.json` file with your dnsdist server configurations:

```json
{
  "servers": [
    {
      "name": "dnsdist-1",
      "address": "127.0.0.1:5199",
      "api_key": "your-api-key-here",
      "timeout": 10000000000
    },
    {
      "name": "dnsdist-2",
      "address": "192.168.1.100:5199",
      "api_key": "your-api-key-here",
      "timeout": 10000000000
    }
  ]
}
```

### Configuration Fields

- `name`: Unique identifier for the server
- `address`: Server address in `host:port` format
- `api_key`: API key for HMAC authentication (leave empty if authentication is not enabled)
- `timeout`: Connection timeout in nanoseconds (default: 10000000000 = 10 seconds)

## Usage

### List Configured Servers

```bash
./dnsdist-worker -config config.json -list
```

### Execute Command on All Servers

```bash
./dnsdist-worker -config config.json -command "showServers()"
```

### Execute Command on Specific Server

```bash
./dnsdist-worker -config config.json -server dnsdist-1 -command "showServers()"
```

### Common dnsdist Commands

- `showServers()`: List all backend servers
- `showRules()`: Display all rules
- `showPools()`: Show server pools
- `getStats()`: Get statistics
- `addPoolRule("example.com", "pool1")`: Add a rule to route queries
- `newServer({address="10.0.0.1:53", pool="pool1"})`: Add a new backend server

## Library Usage

You can also use the dnsdist-worker package in your own Go applications:

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/vitistack/dnsdist-worker/pkg/dnsdist"
)

func main() {
    // Create a new worker
    worker := dnsdist.NewWorker()
    defer worker.Close()
    
    // Add servers
    worker.AddServer(&dnsdist.ServerConfig{
        Name:    "dnsdist-1",
        Address: "127.0.0.1:5199",
        APIKey:  "your-api-key",
        Timeout: 10 * time.Second,
    })
    
    // Connect to all servers
    errors := worker.ConnectAll()
    for name, err := range errors {
        log.Printf("Failed to connect to %s: %v", name, err)
    }
    
    // Execute a command on all servers
    results := worker.ExecuteCommandAll("showServers()")
    for name, result := range results {
        if result.Error != nil {
            log.Printf("%s: Error: %v", name, result.Error)
        } else {
            fmt.Printf("%s: %s\n", name, result.Response)
        }
    }
}
```

## DNSDist Control Socket Protocol

The control socket protocol used by dnsdist is a simple text-based protocol:

1. **Connection**: Client connects to the control socket (default port 5199)
2. **Authentication** (if enabled):
   - Server sends a challenge string
   - Client computes HMAC-SHA256 of the challenge using the API key
   - Client sends the hex-encoded HMAC as response
3. **Command Execution**:
   - Client sends commands as plain text (Lua code)
   - Server executes the command and returns the result
   - Multiple commands can be sent over the same connection

## Development

### Running Tests

```bash
go test ./...
```

### Running Tests with Coverage

```bash
go test -cover ./...
```

### Building

```bash
go build -o dnsdist-worker ./cmd/dnsdist-worker
```

## Security Considerations

- Always use API keys when connecting to dnsdist servers in production
- Keep your API keys secure and never commit them to version control
- Use TLS/SSH tunneling for connections over untrusted networks
- Limit access to the control socket using firewall rules

## License

Apache License 2.0 - See LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## DNSDist Documentation

For more information about dnsdist and its commands, visit:
- [DNSDist Official Documentation](https://dnsdist.org/)
- [DNSDist Control Socket](https://dnsdist.org/guides/console.html)