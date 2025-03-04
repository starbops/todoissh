# TodoiSSH

A terminal-based todo list application accessible via SSH, built with Go. This application allows users to manage their todo items through a clean and interactive terminal user interface.

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

## Prerequisites

- Go 1.x or higher
- SSH client

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/starbops/todoissh.git
   cd todoissh
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the application:
   ```bash
   go build
   ```

## Usage

1. Start the server:
   ```bash
   ./todoissh
   ```
   The server will start listening on port 2222 by default.

2. Connect to the server using SSH:
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
- `.gitignore`: Specifies which files Git should ignore
- `go.mod` & `go.sum`: Go module files for dependency management

## Security Note

This is a demonstration project and includes simplified security measures:
- Accepts any username/password combination
- Generates a new SSH host key if none exists
- Stores todos in memory (data is lost when server restarts)

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

Contributions are welcome! Please feel free to submit a Pull Request. 