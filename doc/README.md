# Stackit Documentation

This directory contains the source for the stackit documentation site, built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/).

## Quick Start

### Prerequisites

- Python 3.12+
- [uv](https://docs.astral.sh/uv/) package manager

### Install Dependencies

```bash
uv sync
```

### Local Development

Serve the documentation locally with live reload:

```bash
uv run mkdocs serve
```

The site will be available at http://localhost:8000

### Build for Production

```bash
uv run mkdocs build
```

The static site will be generated in the `_site/` directory.

## Project Structure

```
doc/
├── mkdocs.yml          # Main configuration
├── pyproject.toml      # Python dependencies
├── hooks/              # Custom MkDocs hooks
│   ├── cliref.py      # CLI reference generator
│   └── replace.py     # Command link syntax ($$ ... $$)
├── includes/           # Reusable snippets
│   └── cli-reference.md  # Generated CLI docs
├── overrides/          # Theme customizations
├── src/                # Documentation source
│   ├── index.md       # Homepage
│   ├── start/         # Getting started guide
│   ├── guide/         # User guide
│   ├── cli/           # CLI reference
│   ├── community/     # Community resources
│   ├── css/           # Custom CSS
│   └── img/           # Images and logo
└── _site/             # Build output (gitignored)
```

## Writing Documentation

### Command Links

Use the `$$...$$ syntax to create links to CLI commands:

- `$$stackit create$$` → Links to CLI reference for `stackit create`
- `$$stackit create|st create$$` → Display as "st create" but link to `stackit create`

### CLI Reference

The CLI reference is auto-generated from `stackit --help` during the build process. To regenerate:

1. Build the stackit binary:
   ```bash
   cd .. && just build
   ```

2. Rebuild the docs:
   ```bash
   uv run mkdocs build
   ```

### Icons

Use Material Design icons in frontmatter:

```markdown
---
icon: material/rocket-launch
---
```

Browse available icons: https://squidfunk.github.io/mkdocs-material/reference/icons-emojis/

## Deployment

The documentation is automatically deployed to GitHub Pages when changes are pushed to the `main` branch. See `.github/workflows/doc.yml` for the deployment configuration.

### Manual Deployment

```bash
uv run mkdocs gh-deploy
```

## Configuration

Key configuration options in `mkdocs.yml`:

- **site_name**: Site title
- **site_url**: Production URL
- **nav**: Navigation structure
- **theme**: Material theme settings
- **plugins**: MkDocs plugins
- **markdown_extensions**: Markdown features

## Tips

- Use admonitions for callouts: `!!! note`, `!!! tip`, `!!! warning`
- Add mermaid diagrams with ` ```mermaid ` code blocks
- Use tabbed content with `=== "Tab Title"`
- Link between pages with relative paths: `[text](../guide/concepts.md)`

## Troubleshooting

### Build fails with "template not found"

Check that all icon references in frontmatter use valid Material icons.

### CLI reference not generated

Ensure the stackit binary exists in the repository root. Build it with `just build`.

### Changes not showing up

Clear the cache and rebuild:

```bash
rm -rf _site/ .cache/
uv run mkdocs build
```

## More Information

- [MkDocs Documentation](https://www.mkdocs.org/)
- [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/)
- [Markdown Extensions](https://squidfunk.github.io/mkdocs-material/reference/)
