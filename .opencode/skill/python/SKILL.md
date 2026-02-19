---
name: "python"
description: "Python development with uv and PEP 723 inline dependencies"
---

# Python Development Best Practices

## Overview

Python scripts in this repository use `uv` for dependency management and follow a standalone script pattern with inline PEP 723 dependency declarations when external packages are needed.

## Always Use uv

- **NEVER** use `pip`, `pipenv`, `poetry`, or `venv` directly
- **ALWAYS** use `uv` for all Python package management
- Script shebang: `#!/usr/bin/env -S uv run --quiet --script`

## Script Templates

### Script with Dependencies

```python
#!/usr/bin/env -S uv run --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "click",
#     "rich",
#     "questionary",
# ]
# ///

"""Brief script description."""

import click
from rich.console import Console

console = Console()

@click.command()
def main():
    """Main entry point."""
    console.print("[green]Hello![/green]")

if __name__ == "__main__":
    main()
```

### Simple Script (No Dependencies)

```python
#!/usr/bin/env python3

"""Brief script description."""

def main():
    """Main entry point."""
    print("Hello!")

if __name__ == "__main__":
    main()
```

## PEP 723 Inline Script Metadata

When external packages are needed, use inline dependency declarations:

```python
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "package-name>=1.0.0",
# ]
# ///
```

**Benefits**:
- No separate requirements.txt needed
- Script is self-contained and portable
- `uv run` automatically installs dependencies in isolated environment

## Common Dependencies

### CLI Tools
```python
"click>=8.1.0",          # Command-line interface
"typer>=0.9.0",          # Alternative CLI framework
"questionary>=2.0.0",    # Interactive prompts
```

### Output & Formatting
```python
"rich>=13.0.0",          # Rich terminal output
"tabulate>=0.9.0",       # Table formatting
```

### Data Handling
```python
"pyyaml>=6.0",           # YAML parsing
"pydantic>=2.0",         # Data validation
"python-dateutil>=2.8",  # Date/time parsing
```

### HTTP & APIs
```python
"httpx>=0.27.0",         # Modern HTTP client
"requests>=2.31.0",      # Traditional HTTP client
```

## Running Scripts

```bash
# Direct execution
./script_name.py

# With arguments
./script_name.py --option value
```

## Anti-Patterns (DO NOT USE)

❌ `pip install`
❌ `python -m venv`
❌ `requirements.txt` for standalone scripts
❌ Global package installs

## Best Practices

✅ Use `uv` for all package management
✅ Use inline `# /// script` block when dependencies needed
✅ Use type hints (Python 3.11+)
✅ Make scripts executable: `chmod +x script.py`
✅ Use `pathlib.Path` instead of string paths
✅ Handle errors with try/except
✅ Use `rich.console` for colored output
✅ Use `click` or `typer` for CLI parsing

## Common Patterns

### CLI Tool
```python
#!/usr/bin/env -S uv run --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["click>=8.1.0", "rich>=13.0.0"]
# ///

import click
from rich.console import Console

console = Console()

@click.command()
@click.option('--verbose', is_flag=True)
@click.argument('name')
def main(verbose: bool, name: str):
    """Greet NAME."""
    if verbose:
        console.print("[dim]Verbose mode[/dim]")
    console.print(f"[green]Hello, {name}![/green]")

if __name__ == "__main__":
    main()
```

### Interactive Prompts
```python
#!/usr/bin/env -S uv run --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["questionary>=2.0.0", "rich>=13.0.0"]
# ///

import questionary
from rich.console import Console

console = Console()

def main():
    """Interactive tool."""
    name = questionary.text("Name?").ask()
    choice = questionary.select(
        "Pick one:",
        choices=["A", "B", "C"]
    ).ask()
    console.print(f"[green]{choice}[/green]")

if __name__ == "__main__":
    main()
```

## When NOT to Use Python

Use bash/fish for:
- Simple file operations
- Git operations
- Single-command wrappers

Use Python for:
- Complex logic
- Data transformation
- API interactions
- Interactive CLI tools
