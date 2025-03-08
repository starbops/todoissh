# TodoiSSH

[![CI](https://github.com/starbops/todoissh/actions/workflows/ci.yml/badge.svg)](https://github.com/starbops/todoissh/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/starbops/todoissh)](https://go.dev)
[![License](https://img.shields.io/github/license/starbops/todoissh)](LICENSE)

A multi-user todo list application accessible via SSH with a clean terminal interface. Create your personal account, manage your tasks from anywhere, and enjoy the security of isolated user data.

## Quick Start

### Using Docker (Recommended)

```bash
# Run the container
docker run -d -p 2222:2222 starbops/todoissh

# Connect with your username
ssh yourname@localhost -p 2222
```

### Running Locally

```bash
# Clone and build
git clone https://github.com/starbops/todoissh.git
cd todoissh
make all

# Start the server
./bin/todoissh

# Connect with your username
ssh yourname@localhost -p 2222
```

## Features

- **Multi-User Support** - Each user has their own private todo list with secure authentication
- **Terminal UI** - Clean, intuitive interface optimized for keyboard navigation
- **Data Persistence** - Your todos are saved between sessions
- **Secure Design** - Password hashing with bcrypt and data isolation between users
- **Accessible Anywhere** - Connect via any SSH client, even on mobile devices

## Usage

### First-Time Connection

When connecting with a new username, you'll be guided through a simple registration:

```
Welcome to TodoiSSH!
────────────────────────────────────────────────────────────────────────────────

Hello, newuser! You need to complete registration.

Please set a password for your account.
Password must be at least 6 characters long.

Password: ******
```

### Managing Your Todos

After authentication, you'll see your personal todo list with full keyboard controls:

```
Todo List - User: myusername
────────────────────────────────────────────────────────────────────────────────
Commands: ↑/↓: Navigate • Space: Toggle • Enter: Edit • Tab: New • Delete: Remove • Ctrl+C: Exit

[ ] Buy groceries
[✓] Finish documentation
[ ] Fix bug in registration flow
[ ] Add unit tests

────────────────────────────────────────────────────────────────────────────────
```

**Keyboard Controls:**
- ↑/↓: Navigate through todos
- Space: Toggle completion status
- Enter: Edit selected todo
- Tab: Create new todo
- Delete: Remove selected todo
- Ctrl+C: Exit application

## Advanced Usage

### Using Docker with Persistent Storage

```bash
# Create a volume for persistent storage
docker volume create todoissh-data

# Run with the volume mounted
docker run -d -p 2222:2222 -v todoissh-data:/app/data todoissh
```

### Custom Configuration

```bash
# Run with a custom port
./bin/todoissh --port 2223

# Enable debug logging
./bin/todoissh --debug
```

## Development

### Project Structure

```
todoissh/
├── main.go              # Application entry point
├── pkg/                 # Application packages
│   ├── config/          # Configuration management
│   ├── ssh/             # SSH server implementation
│   ├── todo/            # Todo list data structure
│   └── ui/              # Terminal user interface
├── test/                # Test files and reports
└── scripts/             # Build and test scripts
```

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/todo
```

See [test/README.md](test/README.md) for detailed testing information.

## Security Note

- User authentication with bcrypt password hashing
- Data isolation between users
- SSH host key generation and management
- Docker container runs as non-root user

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.

## Built With

- Go 1.24+
- [Cursor](https://cursor.sh/) - The AI-first code editor
- Claude 3.5 Sonnet - Advanced AI assistant
