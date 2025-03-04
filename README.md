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
   git clone [your-repository-url]
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

- `main.go`: Contains the SSH server implementation and terminal UI logic
- `todo/todo.go`: Implements the todo list data structure and operations
- `.gitignore`: Specifies which files Git should ignore
- `go.mod` & `go.sum`: Go module files for dependency management

## Security Note

This is a demonstration project and includes simplified security measures:
- Accepts any username/password combination
- Generates a new SSH host key if none exists
- Stores todos in memory (data is lost when server restarts)

For production use, proper authentication and persistent storage should be implemented.

## License

[Your chosen license]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 