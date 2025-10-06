# reqo

**reqo** is a friendly HTTP client that groups requests into projects, supports environments, reusable header sets and saved aliases. Think of it as `curl`, but with project organization and saved calls.

## Features

- 🗂️ **Project-based organization** - Group related API calls into projects
- 🌍 **Environment management** - Switch between dev, staging, production environments
- 📋 **Header sets** - Reusable header configurations (auth tokens, API keys, etc.)
- 💾 **Saved calls** - Create aliases for frequently used requests
- 🔧 **Ad-hoc requests** - Make one-off requests without saving
- 🐚 **Curl export** - Generate equivalent curl commands
- 📝 **Template variables** - Use `${var}` syntax for dynamic values
- 🎨 **Pretty output** - Automatic JSON formatting and colored output

## Installation

### Build from source

```bash
git clone https://github.com/suprbdev/reqo.git
cd reqo
make build    # Build for current platform
# or
make dev      # Build for development (current directory)
```

### Install globally

```bash
go install github.com/suprbdev/reqo/cmd/reqo@latest  # Install to home from GitHub
make install                                         # Install to GOBIN or GOPATH/bin
make install-system                                  # Install to /usr/local/bin (requires sudo)
```

### Cross-platform builds

```bash
make build-all      # Build for all platforms (Linux, macOS, Windows)
make build-linux    # Build for Linux
make build-darwin   # Build for macOS (Intel + Apple Silicon)
make build-windows  # Build for Windows
```

### Development

```bash
make dev-setup      # Setup development environment (tidy, fmt, lint)
make test           # Run tests
make test-coverage  # Run tests with coverage report
make lint           # Run linter
make fmt            # Format code
make tidy           # Tidy dependencies
make clean          # Clean build artifacts
make help           # Show all available targets
```

## Quick Start

1. **Build and install reqo:**
   ```bash
   git clone https://github.com/suprbdev/reqo.git
   cd reqo
   make build && make install
   ```

2. **Initialize a project:**
   ```bash
   reqo init my-api
   ```

3. **Add an environment:**
   ```bash
   reqo env add prod --base-url https://api.example.com
   ```

4. **Create a saved call:**
   ```bash
   reqo call create get-users GET /users
   ```

5. **Execute the call:**
   ```bash
   reqo call run get-users --env prod
   ```

6. **Make an ad-hoc request:**
   ```bash
   reqo req GET https://httpbin.org/get
   ```

## Commands

### Project Management

#### `reqo init <project-name>`
Create a new reqo project.

```bash
reqo init my-api                    # Local project
reqo init my-api --global          # Global project (~/.reqo/projects/)
```

#### `reqo use <project-name>`
Set the active project for the current directory.

```bash
reqo use my-api
```

### Environment Management

#### `reqo env add <name> --base-url <url>`
Add a new environment to the current project.

```bash
reqo env add dev --base-url https://dev-api.example.com
reqo env add prod --base-url https://api.example.com
```

#### `reqo env list`
List all environments in the current project.

```bash
reqo env list
```

### Header Management

#### `reqo header set --name <set> "Header: Value"`
Create or update a header set.

```bash
reqo header set --name auth "Authorization: Bearer token123"
reqo header set --name api "X-API-Key: abc123" "Content-Type: application/json"
```

#### `reqo header list`
List all header sets.

```bash
reqo header list
```

#### `reqo header rm <set>`
Remove a header set.

```bash
reqo header rm auth
```

### Saved Calls

#### `reqo call create <alias> <method> <path>`
Create a saved call (alias).

```bash
reqo call create get-users GET /users
reqo call create create-user POST /users --use-headers auth
reqo call create update-user PUT /users/${id} --desc "Update user by ID"

# Use @ for base URL only (useful for GraphQL APIs)
reqo call create graphql-query POST @ --json '{"query": "${query}"}'

# Save calls with request payloads
reqo call create create-user POST /users --json '{"name": "${name}", "email": "${email}"}'
reqo call create upload-file POST /upload --form "file=@${file_path}" --form "description=${desc}"
reqo call create update-config PUT /config --data '{"setting": "${value}"}'
```

#### `reqo call list`
List all saved calls in the current project.

```bash
reqo call list
```

#### `reqo call rm <alias>`
Remove a saved call.

```bash
reqo call rm old-call
```

#### `reqo call run <alias>`
Execute a saved call with optional overrides.

```bash
reqo call run get-users
reqo call run get-users --env prod
reqo call run get-users --header "X-Custom: value"
reqo call run get-users --var id=123
reqo call run get-users --as-curl

# Run calls with saved payloads and variable expansion
reqo call run create-user --var name="John Doe" --var email="john@example.com"
reqo call run upload-file --var file_path="/path/to/file.txt" --var desc="My file"
reqo call run update-config --var value="production"
```

### Ad-hoc Requests

#### `reqo req <method|path> [path]`
Make an ad-hoc HTTP request without saving.

```bash
reqo req GET https://httpbin.org/get
reqo req POST /users --json '{"name": "John"}'
reqo req PUT /users/123 --data '{"name": "Jane"}'
```

### Configuration

#### `reqo config set <key> <value>`
Set a global configuration value.

```bash
reqo config set default_timeout 60
```

#### `reqo config get <key>`
Get a global configuration value.

```bash
reqo config get default_timeout
```

## Project Structure

When you run `reqo init`, it creates a `.reqo/` directory with:

```
.reqo/
├── project.yaml    # Project configuration
└── current         # Active project name
```

### project.yaml Example

```yaml
version: 1
name: my-api
default_env: dev
environments:
  dev:
    base_url: https://dev-api.example.com
  prod:
    base_url: https://api.example.com
header_sets:
  auth:
    - "Authorization: Bearer ${TOKEN}"
  api:
    - "X-API-Key: ${API_KEY}"
    - "Content-Type: application/json"
calls:
  get-users:
    method: GET
    path: /users
    uses_header_set: auth
    description: "Get all users"
  create-user:
    method: POST
    path: /users
    uses_header_set: api
    body:
      json: '{"name": "${name}", "email": "${email}"}'
    description: "Create a new user"
  upload-file:
    method: POST
    path: /upload
    uses_header_set: api
    body:
      form:
        file: "@${file_path}"
        description: "${desc}"
    description: "Upload a file"
  graphql-query:
    method: POST
    path: "@"
    uses_header_set: api
    body:
      json: '{"query": "${query}", "variables": "${variables}"}'
    description: "GraphQL query"
```

## Template Variables

Use `${variable}` syntax for dynamic values:

- **Command-line variables:** `--var key=value`
- **Environment variables:** `${HOME}`, `${USER}`, etc.
- **Project variables:** (future feature)

```bash
reqo run get-user --var id=123
reqo req GET /users/${USER_ID}
```

## Request Payloads

### Base URL Only (GraphQL Support)

For APIs that use a single endpoint (like GraphQL), use `@` as the path to use just the base URL:

```bash
# GraphQL API example
reqo call create graphql-query POST @ --json '{"query": "${query}", "variables": "${variables}"}'
reqo call run graphql-query --var query="query { users { name } }"
```

### Saving Payloads with Calls

You can save request payloads (JSON, raw data, or form fields) with your saved calls:

```bash
# Save a call with JSON payload
reqo call create create-user POST /users --json '{"name": "${name}", "email": "${email}"}'

# Save a call with raw data payload
reqo call create update-config PUT /config --data '{"setting": "${value}"}'

# Save a call with form data
reqo call create upload-file POST /upload --form "file=@${file_path}" --form "description=${desc}"
```

### Variable Expansion in Payloads

When running saved calls, variables in payloads are automatically expanded:

```bash
# The saved JSON payload will have variables replaced
reqo call run create-user --var name="John Doe" --var email="john@example.com"
# Results in: {"name": "John Doe", "email": "john@example.com"}

# Form fields with variables
reqo call run upload-file --var file_path="/path/to/file.txt" --var desc="My file"
# Results in: file=@/path/to/file.txt, description=My file

# Variables in JSON files are also expanded
reqo call run create-user --json @user-template.json --var name="Jane" --var email="jane@example.com"
# If user-template.json contains: {"name": "${name}", "email": "${email}"}
# Results in: {"name": "Jane", "email": "jane@example.com"}
```

### Overriding Saved Payloads

Command-line flags always override saved payloads:

```bash
# This will use the command-line JSON instead of the saved one
reqo call run create-user --json '{"name": "Override", "email": "override@example.com"}'
```

## Request Options

### Data Options
- `--json <data|@file>` - JSON request body (variables in files are expanded)
- `--data <data|@file>` - Raw request body (variables in files are expanded)
- `--form <key=value>` - Multipart form data

### Request Options
- `--header "Key: Value"` - Add headers
- `--query "key=value"` - Add query parameters
- `--env <name>` - Use specific environment
- `--timeout <seconds>` - Request timeout (default: 30)
- `--retries <count>` - Retry count (default: 0)

### Output Options
- `--include` / `-i` - Show response headers
- `--raw` - Raw response body (no formatting)
- `--as-curl` - Print equivalent curl command

## Examples

### API Testing Workflow

```bash
# 1. Initialize project
reqo init my-api

# 2. Add environments
reqo env add dev --base-url https://dev-api.example.com
reqo env add prod --base-url https://api.example.com

# 3. Set up authentication
reqo header set --name auth "Authorization: Bearer ${API_TOKEN}"

# 4. Create saved calls
reqo call create list-users GET /users --use-headers auth
reqo call create create-user POST /users --use-headers auth --json '{"name": "${name}", "email": "${email}"}'
reqo call create get-user GET /users/${id} --use-headers auth

# 5. Test the API
reqo call run list-users --env dev
reqo call run create-user --env dev --var name="John" --var email="john@example.com"
reqo call run get-user --env dev --var id=123

# 6. Generate curl for debugging
reqo call run list-users --env prod --as-curl
```

### One-off Requests

```bash
# Test external APIs
reqo req GET https://httpbin.org/get
reqo req POST https://httpbin.org/post --json '{"test": "data"}'

# Test with custom headers
reqo req GET https://api.github.com/user --header "Authorization: token ${GITHUB_TOKEN}"

# Test with form data
reqo req POST https://httpbin.org/post --form "name=John" --form "email=john@example.com"
```

## Global Configuration

Global configuration is stored in `~/.reqo/config.yaml`:

```yaml
default_timeout: 60
default_retries: 3
```

## Development

### Makefile Features

The project includes a comprehensive Makefile with the following targets:

#### Building
- `make build` - Build binary for current platform
- `make dev` - Build binary for development (current directory)
- `make build-all` - Build for all platforms (Linux, macOS, Windows)
- `make build-linux` - Build for Linux (amd64)
- `make build-darwin` - Build for macOS (Intel + Apple Silicon)
- `make build-windows` - Build for Windows (amd64)

#### Installation
- `make install` - Install to GOBIN or GOPATH/bin
- `make install-system` - Install to /usr/local/bin (requires sudo)

#### Development Tools
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report
- `make bench` - Run benchmarks
- `make lint` - Run linter (golangci-lint or go vet)
- `make fmt` - Format code (go fmt + goimports)
- `make tidy` - Tidy dependencies
- `make clean` - Clean build artifacts

#### Workflow
- `make dev-setup` - Setup development environment (tidy, fmt, lint)
- `make release-prep` - Prepare for release (clean, test, lint, build-all)
- `make run` - Build and run the application
- `make help` - Show all available targets

### Build Variables

You can customize the build with these variables:

```bash
make build BINARY_NAME=my-reqo BUILD_DIR=dist
```

Available variables:
- `BINARY_NAME` - Name of the binary (default: reqo)
- `BUILD_DIR` - Build directory (default: build)
- `VERSION` - Version from git tags (default: dev)

The makefile automatically detects your Go environment:
- Uses `GOBIN` if set, otherwise falls back to `GOPATH/bin`
- Uses `go env` to get current Go settings

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Setup development environment: `make dev-setup`
4. Make your changes
5. Run tests: `make test`
6. Run linter: `make lint`
7. Format code: `make fmt`
8. Add tests if applicable
9. Commit your changes: `git commit -m 'Add amazing feature'`
10. Push to the branch: `git push origin feature/amazing-feature`
11. Submit a pull request

### Development Requirements

- Go 1.22 or later
- Git
- Make (optional, for using the Makefile)

### Optional Tools

For enhanced development experience, install these tools:

```bash
# Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Code formatting
go install golang.org/x/tools/cmd/goimports@latest
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [Viper](https://github.com/spf13/viper) for configuration management
- JSON processing with [gojq](https://github.com/itchyny/gojq)
