# HTTP Proxy Server

A high-performance, rule-based HTTP proxy server written in Go that provides advanced request filtering, logging, and monitoring capabilities.

## Features

- **Rule-Based Filtering**: Support for IPv4/IPv6, URL, Domain, User-Agent, URI suffix, and size-based filtering
- **Dynamic Rule Management**: Hot-reload rules from configuration files or via REST API
- **Multiple Configuration Formats**: YAML, JSON, and TOML support
- **Comprehensive Logging**: Structured logging with audit trails and log rotation
- **Rate Limiting**: Per-IP rate limiting with configurable burst sizes
- **Health Checking**: Backend server health monitoring
- **Statistics & Monitoring**: Real-time proxy statistics and performance metrics
- **Traffic Generation**: Built-in load testing tool for performance validation

## Architecture

```
Client -> HTTP Proxy -> Rule Engine -> Backend Server
           |              |
           |              +-> Block/Allow Decision
           |
           +-> Logging & Statistics
```

## Quick Start

### 🚀 **Option 1: Complete Automated Demo**

```bash
# Run the full functional demo + tests (recommended)
powershell -ExecutionPolicy Bypass -File full_demo.ps1

# Or run with options:
powershell -ExecutionPolicy Bypass -File full_demo.ps1 -QuickTest  # No pauses
powershell -ExecutionPolicy Bypass -File full_demo.ps1 -SkipBuild  # Use existing binaries
```

**What the demo shows:**
- ✅ Complete build process and server startup
- ✅ Real request filtering (allow/block scenarios)  
- ✅ API-based rule management
- ✅ Load testing with statistics
- ✅ Comprehensive unit test suite (69 tests)

### 🛠️ **Option 2: Manual Setup**

### 1. Build the Applications

```bash
# Build proxy server
go build -o proxy cmd/proxy/main.go

# Build backend server (for testing)
go build -o backend cmd/backend/main.go

# Build traffic generator
go build -o traffic-gen cmd/traffic-gen/main.go
```

### 2. Generate Sample Configuration

```bash
go run cmd/config-gen/main.go
```

This creates sample configuration files in the `examples/` directory.

### 3. Start the Backend Server

```bash
./backend
```

The backend server starts on port 8090 by default and provides various endpoints for testing.

### 4. Start the Proxy Server

```bash
# Use default configuration
./proxy

# Use custom configuration
./proxy -config examples/proxy.yaml

# Override specific settings
./proxy -config examples/proxy.yaml -port 9000 -log-level debug
```

### 5. Test the Proxy

```bash
# Test basic functionality
curl -v http://localhost:8080/

# Test blocked requests (admin path)
curl -v http://localhost:8080/admin

# Check proxy statistics
curl http://localhost:8080/proxy/stats

# View proxy health
curl http://localhost:8080/proxy/health
```

## Configuration

### Main Configuration File

The proxy server supports YAML, JSON, and TOML configuration formats:

```yaml
server:
  host: localhost
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

backend:
  host: localhost
  port: 8090
  timeout: 30s
  health_check:
    enabled: true
    interval: 30s
    path: /health

rules:
  default_action: allow
  watch_rules_file: true
  reload_interval: 5s
  rules:
    - id: block-admin
      name: Block Admin Access
      type: url
      operator: starts_with
      value: /admin
      action: block
      priority: 100
      enabled: true

logging:
  level: info
  file: logs/proxy.log
  max_size: 100
  audit_enabled: true
```

### Rule Types and Operations

| Rule Type | Description | Supported Operators |
|-----------|-------------|-------------------|
| `ipv4` | IPv4 address filtering | `equals`, `in_range` |
| `ipv6` | IPv6 address filtering | `equals`, `in_range` |
| `url` | URL path filtering | `equals`, `contains`, `starts_with`, `ends_with`, `wildcard`, `regex` |
| `domain` | Domain name filtering | `equals`, `contains`, `starts_with`, `ends_with`, `wildcard`, `regex` |
| `user_agent` | User-Agent header filtering | `equals`, `contains`, `starts_with`, `ends_with`, `regex` |
| `uri_suffix` | URI suffix filtering | `equals`, `wildcard`, `regex` |
| `size` | Request size filtering | `gte`, `lte`, `in_range`, `equals` |
| `method` | HTTP method filtering | `equals`, `contains` |
| `header` | HTTP header filtering | `equals`, `contains`, `starts_with`, `ends_with`, `regex` |

### Example Rules

```yaml
rules:
  # Block admin endpoints
  - id: block-admin-paths
    name: Block admin paths
    type: url
    operator: starts_with
    value: /admin
    action: block
    priority: 100
    enabled: true

  # Block large uploads
  - id: block-large-requests
    name: Block large requests
    type: size
    operator: gte
    min_size: 10485760  # 10MB
    action: block
    priority: 200
    enabled: true

  # Block suspicious user agents
  - id: block-bots
    name: Block bot traffic
    type: user_agent
    operator: regex
    value: (?i)(bot|crawler|spider|scraper)
    action: block
    priority: 300
    enabled: true

  # Block private IP ranges
  - id: block-private-ips
    name: Block private networks
    type: ipv4
    operator: in_range
    value: 192.168.0.0/16
    action: block
    priority: 150
    enabled: false
```

## API Endpoints

### Proxy Management

- `GET /proxy/health` - Proxy health status
- `GET /proxy/stats` - Proxy statistics

### Rules Management

- `GET /proxy/rules` - List all rules
- `GET /proxy/rules/{id}` - Get specific rule
- `POST /proxy/rules` - Add new rule
- `PUT /proxy/rules/{id}` - Update existing rule
- `DELETE /proxy/rules/{id}` - Delete rule
- `PATCH /proxy/rules/{id}?action=enable|disable` - Enable/disable rule

### Example API Usage

```bash
# List all rules
curl http://localhost:8080/proxy/rules

# Add a new rule
curl -X POST http://localhost:8080/proxy/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "new-rule",
    "name": "Block uploads",
    "type": "url",
    "operator": "starts_with",
    "value": "/upload",
    "action": "block",
    "priority": 400,
    "enabled": true
  }'

# Disable a rule
curl -X PATCH "http://localhost:8080/proxy/rules/new-rule?action=disable"

# Delete a rule
curl -X DELETE http://localhost:8080/proxy/rules/new-rule
```

## Traffic Generation

Use the built-in traffic generator to test proxy performance:

```bash
# Basic load test
./traffic-gen -proxy http://localhost:8080 -c 20 -d 5m -rps 50

# Test with custom scenarios
./traffic-gen -scenarios examples/scenarios.json -save

# High load test
./traffic-gen -c 100 -rps 1000 -d 10m
```

### Custom Scenarios

Create a JSON file with custom test scenarios:

```json
[
  {
    "name": "normal_request",
    "weight": 70,
    "method": "GET",
    "path": "/api/data"
  },
  {
    "name": "admin_attempt",
    "weight": 20,
    "method": "GET",
    "path": "/admin",
    "user_agent": "BadBot/1.0"
  },
  {
    "name": "large_upload",
    "weight": 10,
    "method": "POST",
    "path": "/upload",
    "body_size": 1048576
  }
]
```

## Testing & Quality Assurance

### Automated Test Suite

Run the comprehensive test suite:

```bash
# PowerShell test runner with coverage reporting
powershell -ExecutionPolicy Bypass -File run_tests.ps1

# Or run tests manually
go test ./internal/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Test Coverage Summary

| Component | Tests | Coverage | Features Tested |
|-----------|-------|----------|----------------|
| Rules Engine | 18 tests | ~81% | Pattern matching, IP filtering, rule evaluation |
| Configuration | 11 tests | ~82% | Multi-format parsing, validation, defaults |
| Rate Limiter | 13 tests | ~65% | Token bucket algorithm, concurrency |
| Logging System | 12 tests | ~70% | Structured logging, audit trails |
| **Total** | **54 tests** | **~75%** | **All critical business logic** |

## Environment Variables

Configure the proxy using environment variables:

```bash
# Proxy configuration
export PROXY_CONFIG_FILE=config/proxy.yaml
export PROXY_PORT=8080
export PROXY_LOG_LEVEL=debug

# Backend server configuration
export BACKEND_PORT=8090
export BACKEND_DELAY_MIN=0
export BACKEND_DELAY_MAX=100ms
export BACKEND_ERROR_RATE=0.05
export BACKEND_ENABLE_LOGGING=true
```

## Logging and Monitoring

### Log Levels

- `debug`: Detailed debugging information
- `info`: General information about proxy operations
- `warn`: Warning messages
- `error`: Error messages

### Audit Logging

The proxy maintains detailed audit logs in JSON format:

```json
{
  "timestamp": "2023-11-26T10:00:00Z",
  "request_id": "1234567890-1",
  "client_ip": "192.168.1.100",
  "method": "GET",
  "url": "/admin",
  "user_agent": "Mozilla/5.0...",
  "request_size": 0,
  "rule_matched": "block-admin-paths",
  "action": "block",
  "reason": "URL '/admin' starts with '/admin'",
  "duration_ms": 15,
  "response_code": 403
}
```

### Statistics

Monitor proxy performance through the statistics endpoint:

```json
{
  "total_requests": 12345,
  "allowed_requests": 11000,
  "blocked_requests": 1200,
  "error_requests": 145,
  "average_latency_ms": 25,
  "rules_evaluated": 12345
}
```

## Performance Tuning

### Rate Limiting

Configure per-IP rate limiting:

```yaml
security:
  rate_limiting:
    enabled: true
    requests_per_sec: 100
    burst_size: 10
    cleanup_interval: 60s
```

### Server Timeouts

Adjust timeouts for optimal performance:

```yaml
server:
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  max_header_bytes: 1048576  # 1MB
```

### Backend Configuration

Configure backend connection settings:

```yaml
backend:
  timeout: 30s
  health_check:
    enabled: true
    interval: 30s
    timeout: 5s
```

## Development

### Project Structure

```
├── cmd/
│   ├── proxy/          # Main proxy application
│   ├── backend/        # Test backend server
│   ├── traffic-gen/    # Traffic generation tool
│   └── config-gen/     # Configuration generator
├── internal/
│   ├── proxy/          # Proxy core functionality
│   ├── rules/          # Rule engine
│   ├── config/         # Configuration management
│   └── logger/         # Logging system
├── pkg/
│   └── types/          # Shared type definitions
└── examples/           # Sample configurations and scenarios
```

### Building from Source

```bash
# Clone repository
git clone <repository-url>
cd http-proxy

# Install dependencies
go mod download

# Build all applications
make build

# Run comprehensive unit tests  
make test

# Run tests with coverage report
make coverage

# Run individual component tests
go test ./internal/config/ -v    # Configuration tests
go test ./internal/rules/ -v     # Rules engine tests  
go test ./internal/proxy/ -v     # Proxy & rate limiter tests
go test ./internal/logger/ -v    # Logging system tests
```

### Testing Strategy

The project follows Go testing best practices with comprehensive unit tests for all business logic:

```
Component Coverage:
├── internal/config/    ~82% - Configuration parsing & validation
├── internal/rules/     ~81% - Rule engine & pattern matching  
├── internal/proxy/     ~65% - Rate limiting & proxy logic
├── internal/logger/    ~70% - Logging & audit system
└── Total:              ~75% - 69 comprehensive unit tests

Entry Points (No Tests Needed):
├── cmd/proxy/          Main application (wires tested components)
├── cmd/backend/        Test utility server
├── cmd/traffic-gen/    Load testing tool
└── cmd/config-gen/     Configuration generator
```

**Why cmd/ packages don't need tests**: These are entry points that simply wire together already-tested internal components. The business logic is thoroughly tested in the `internal/` packages.

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o proxy cmd/proxy/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/proxy .
COPY --from=builder /app/examples/ ./examples/

CMD ["./proxy", "-config", "examples/proxy.yaml"]
```

### Systemd Service

```ini
[Unit]
Description=HTTP Proxy Server
After=network.target

[Service]
Type=simple
User=proxy
Group=proxy
WorkingDirectory=/opt/proxy
ExecStart=/opt/proxy/proxy -config /etc/proxy/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Security Considerations

1. **Access Control**: Use rule-based filtering to control access to sensitive endpoints
2. **Rate Limiting**: Implement rate limiting to prevent abuse
3. **Logging**: Enable comprehensive audit logging for security monitoring
4. **Network Security**: Deploy behind a firewall and use HTTPS when possible
5. **Configuration Security**: Secure configuration files and API endpoints

## Troubleshooting

### Common Issues

1. **High Memory Usage**: Check rule complexity and audit log retention
2. **Connection Timeouts**: Adjust timeout settings for backend and client connections
3. **Rule Not Matching**: Verify rule syntax and test with debug logging
4. **Performance Issues**: Monitor statistics and consider rate limiting

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
./proxy -log-level debug
```

### Documentation Files

- **README.md** - Complete project documentation
- **DEMO_WALKTHROUGH.md** - Step-by-step functional demo guide
- **TEST_SUMMARY.md** - Detailed unit test documentation  
- **full_demo.ps1** - Complete automated demo (functional + tests)
- **demo.ps1** - Interactive demonstration script
- **run_tests.ps1** - Automated test runner with coverage
- **examples/** - Sample configurations and scenarios

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- All business logic must be in `internal/` packages with unit tests
- Maintain >75% test coverage on core components  
- Entry points in `cmd/` are integration points only
- Follow Go testing best practices
- Update documentation for new features

## License

This project is licensed under the MIT License - see the LICENSE file for details.
