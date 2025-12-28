# Contributing to Stackit

Thank you for your interest in contributing to Stackit! This document outlines the process for contributing to the project.

## Getting Started

### Prerequisites

- **Go 1.25+**
- **Git 2.25+**
- **GitHub CLI (`gh`)** for PR operations
- **[just](https://github.com/casey/just)** (optional, but recommended for running development commands)

### Setting Up Your Development Environment

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/your-username/stackit.git
   cd stackit
   ```

2. Install dependencies:
   ```bash
   just deps
   # Or manually:
   go mod download
   ```

3. Build the project:
   ```bash
   just build
   ```

4. Initialize stackit in the repository:
   ```bash
   just init
   # Or: ./stackit init
   ```

## Development Workflow

### Using Stackit for Contributions

**All contributions must use Stackit to manage branches and commits.** This ensures consistency with the project's workflow and helps maintain a clean Git history.

1. Create a new branch for your changes:
   ```bash
   stackit create your-feature-name -m "feat: add your feature"
   ```

2. Make your changes and commit them:
   ```bash
   git add .
   stackit modify -m "feat: implement feature details"
   ```

3. If you need multiple related changes, stack them:
   ```bash
   # After making more changes
   stackit create additional-changes -m "feat: add more functionality"
   ```

4. View your stack:
   ```bash
   stackit log
   ```

5. Submit your changes:
   ```bash
   stackit submit
   ```

### Running Tests and Linting

Before submitting your changes, ensure all tests pass and the code is properly formatted:

```bash
# Run all checks (format, lint, test)
just check

# Or run individually:
just fmt      # Format code
just lint     # Run linter
just test     # Run tests
```

All changes must pass tests and linting before being submitted.

## Commit Message Format

**Stackit uses [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages.**

### Format

```
<type>[optional scope]: <description>
```

### Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, missing semi-colons, etc.)
- `refactor`: Code refactoring without changing functionality
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks, dependency updates, etc.
- `ci`: Changes to CI configuration

### Examples

```
feat: add branch traversal functionality
fix: resolve merge conflict detection issue
refactor: simplify merge plan logic
docs: update README with new command examples
test: add tests for branch operations
chore: update dependencies
```

### Best Practices

- Use the imperative mood ("add" not "added" or "adds")
- Keep the description concise but descriptive
- Reference issues or PRs when applicable: `feat: add feature (#123)`
- Use the scope when it helps clarify the change: `feat(engine): add new merge strategy`

## Submitting Changes

1. Ensure your code passes all checks:
   ```bash
   just check
   ```

2. Submit your stack to GitHub:
   ```bash
   stackit submit
   ```

3. Create a Pull Request on GitHub with a clear description of your changes.

4. Ensure your PR description follows the same conventions as commit messages when possible.

## Code Style

- Follow Go standard formatting (use `just fmt` or `goimports`)
- Follow the existing code style in the repository
- Write clear, self-documenting code
- Add comments for complex logic

## Questions?

If you have questions about contributing, please open an issue on GitHub or reach out to the maintainers.

Thank you for contributing to Stackit! 🎉
