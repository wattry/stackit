"""
Generate config reference documentation from stackit docs config output.

This hook runs during the build process and generates the config-reference.md
file in the includes directory based on the config metadata registry.
"""

import subprocess
from pathlib import Path
from mkdocs.config.defaults import MkDocsConfig


def on_pre_build(config: MkDocsConfig):
    """Generate config reference documentation before building the site."""
    doc_dir = Path(config.docs_dir).parent
    repo_root = doc_dir.parent

    stackit_bin = repo_root / "stackit"

    if not stackit_bin.exists():
        print(f"stackit binary not found at {stackit_bin}")
        print("Run 'mise run build' first, then rebuild docs")
        _create_placeholder(doc_dir)
        return

    try:
        # Generate config reference
        result = subprocess.run(
            [str(stackit_bin), "docs", "config"],
            capture_output=True,
            text=True,
            check=True
        )
        config_docs = result.stdout

        # Generate YAML example
        yaml_result = subprocess.run(
            [str(stackit_bin), "docs", "yaml-example"],
            capture_output=True,
            text=True,
            check=True
        )
        yaml_example = yaml_result.stdout

        # Build the complete reference
        output = _build_config_reference(config_docs, yaml_example)

        # Write to includes directory
        includes_dir = doc_dir / "includes"
        includes_dir.mkdir(exist_ok=True)

        output_file = includes_dir / "config-reference.md"
        output_file.write_text(output)

        print(f"Generated config reference at {output_file}")

    except subprocess.CalledProcessError as e:
        print(f"Failed to generate config reference: {e}")
        print(f"stderr: {e.stderr}")
        _create_placeholder(doc_dir)
    except Exception as e:
        print(f"Unexpected error generating config reference: {e}")
        _create_placeholder(doc_dir)


def _build_config_reference(config_docs: str, yaml_example: str) -> str:
    """Build the complete config reference document."""
    output = []

    output.append(config_docs)

    output.append("## Team configuration (`.stackit.yaml`)")
    output.append("")
    output.append("For team-wide settings that should be shared across all contributors, create a `.stackit.yaml` file in your repository root and commit it to version control. Team settings act as defaults that individual developers can override in their personal git config.")
    output.append("")
    output.append("See the [Team Collaboration Guide](../workflows/collaboration.md) for collaboration patterns using shared configuration.")
    output.append("")
    output.append("```yaml")
    output.append(yaml_example)
    output.append("```")
    output.append("")

    return '\n'.join(output)


def _create_placeholder(doc_dir: Path):
    """Create a placeholder config reference file."""
    includes_dir = doc_dir / "includes"
    includes_dir.mkdir(exist_ok=True)

    output_file = includes_dir / "config-reference.md"
    placeholder = """!!! warning "Config reference not available"

    The config reference could not be generated automatically.
    Please build the stackit binary and rebuild the documentation.

    ```bash
    mise run build
    cd doc && uv run mkdocs build
    ```

For a complete list of configuration options, run:

```bash
stackit config --list
```
"""
    output_file.write_text(placeholder)
    print(f"Created placeholder config reference at {output_file}")
