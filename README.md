# TodoiSSH

[![CI](https://github.com/starbops/todoissh/actions/workflows/ci.yml/badge.svg)](https://github.com/starbops/todoissh/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/starbops/todoissh)](https://go.dev)
[![License](https://img.shields.io/github/license/starbops/todoissh)](LICENSE)

A terminal-based todo list application accessible via SSH, built with Go. This application allows users to manage their todo items through a clean and interactive terminal user interface.

## Built With

This project was created through an innovative collaboration:
- Human guidance and vision
- [Cursor](https://cursor.sh/) - The AI-first code editor
- Claude 3.5 Sonnet - Advanced AI assistant

## Features

- **SSH Access**: Connect to your todo list from anywhere using SSH
- **Interactive TUI**: Clean and responsive terminal user interface
- **Todo Management**:
  - View todo items in a list
  - Add new todo items
  - Edit existing todos
  - Toggle completion status
  - Delete todos
- **Keyboard Navigation**:
  - ↑/↓: Navigate through todos
  - Space: Toggle completion status
  - Enter: Edit selected todo
  - Tab: Create new todo
  - Delete: Remove selected todo
  - Ctrl+C: Exit application
- **Docker Support**: Run in an isolated container for improved security
- **Continuous Integration**: Automated testing and building via GitHub Actions

## Prerequisites

- Go 1.24 or higher
- SSH client
- Docker (optional, for containerized deployment)
- Make

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/starbops/todoissh.git
   cd todoissh
   ```

2. Build and test:
   ```bash
   make all
   ```
   This will:
   - Run all tests
   - Generate test reports
   - Build the binary
   - Create a Docker image

   Individual commands:
   - `make test`: Run tests only
   - `make build`: Build the binary
   - `make package`: Create Docker image
   - `make clean`: Clean up build artifacts

## Usage

### Running Locally

1. Start the server:
   ```bash
   ./bin/todoissh
   ```
   The server will start listening on port 2222 by default.

### Running with Docker

1. Start the container:
   ```bash
   docker run -d -p 2222:2222 todoissh
   ```

2. Connect to the server:
   ```bash
   ssh localhost -p 2222
   ```
   Note: The server accepts any username/password combination for demonstration purposes.

## Development

The project structure is organized as follows:

- `main.go`: Application entry point
- `pkg/`: Application packages
  - `config/`: Configuration management
  - `ssh/`: SSH server implementation
  - `todo/`: Todo list data structure and operations
  - `ui/`: Terminal user interface implementation
- `scripts/`: Build and test scripts
  - `build`: Builds the application
  - `test`: Runs tests and generates reports
  - `package`: Creates Docker image
- `bin/`: Build artifacts (not in git)
- `test/reports/`: Test results and coverage reports (not in git)
- `.github/workflows/`: CI pipeline definitions
- `Dockerfile`: Container definition
- `Makefile`: Build automation
- `.gitignore`: Specifies which files Git should ignore
- `go.mod` & `go.sum`: Go module files for dependency management

## Testing

Run the test suite:
```bash
make test
```

Test artifacts will be generated in `test/reports/`:
- `coverage.html`: HTML coverage report
- `test-report.json`: Detailed test results
- `test-summary.txt`: Human-readable test summary

## Security Note

This is a demonstration project and includes simplified security measures:
- Accepts any username/password combination
- Generates a new SSH host key if none exists
- Stores todos in memory (data is lost when server restarts)
- When using Docker, runs as non-root user in an isolated container

For production use, proper authentication and persistent storage should be implemented.

## License

Apache License 2.0

Copyright 2024 Zespre Chang <starbops@zespre.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. The project uses GitHub Actions for CI, ensuring all tests pass before merging. 