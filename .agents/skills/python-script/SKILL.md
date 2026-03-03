---
name: python-script
description: Creates standalone Python scripts using uv with PEP 723 inline script metadata.
---

# Python Script with uv

## Template

```python
#!/usr/bin/env -S uv run --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["dep1", "dep2"]
# ///
"""Brief description."""

import argparse


def main():
    parser = argparse.ArgumentParser(description="Description")
    # args...
    args = parser.parse_args()
    # implementation...


if __name__ == "__main__":
    main()
```

## Rules

- Shebang: `#!/usr/bin/env -S uv run --quiet --script`
- PEP 723 block with `requires-python` and `dependencies`
- Only list external dependencies (not stdlib)
- Use `argparse` for CLI
- Always use `if __name__ == "__main__": main()`
