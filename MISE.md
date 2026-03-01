# Using mise

Stackit uses [mise](https://mise.jdx.dev/) for tool version management and task running.

## Quick Start

```bash
# Install mise
brew install mise
# or: curl https://mise.run | sh

# Activate mise in your shell
echo 'eval "$(mise activate bash)"' >> ~/.bashrc   # or ~/.zshrc for zsh

# Install all project tools
cd /path/to/stackit
mise install
```

## Common Commands

```bash
# View all available tasks
mise tasks

# Run tasks
mise run check         # Format, lint, and test
mise run build         # Build the binary
mise run test          # Run all tests
mise run test:fast     # Run fast unit tests
mise run lint          # Run linter
mise run fmt           # Format code

# Install/update tools
mise install           # Install tools from mise.toml
mise upgrade           # Upgrade tools to latest versions
```

## How It Works

- `mise.toml` defines tools (Go, gotestsum, golangci-lint, ripgrep, fd, etc.) and tasks (build, test, lint)
- Tools are automatically available in your PATH when in the project directory
- Environment variables like `GOBIN` and `PROJECT_ROOT` are set automatically

## Learn More

- [mise documentation](https://mise.jdx.dev/)
