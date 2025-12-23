---
triggers: [config, configuration, viper, yaml, settings, precedence]
---

# Config System

## Precedence (highest wins)

1. `CLY_*` env vars
2. `config.local.yaml` (gitignored, dev secrets)
3. `~/.config/cly/config.yaml` (user overrides)
4. Embedded defaults (in binary)

## Secrets

Use `op://vault/item/field` in `modules.*` config.
Resolved at load via 1Password CLI.

## Structure

```yaml
app:
  dotfiles_dir: ~/DotFiles
modules:
  bundle:
    go_file: ~/.config/Gofile
  scraper:
    browser:
      debug_port: 9222
```

## Code

- pkg/config/config.go:102-152 - loading
- pkg/config/secrets.go:26-46 - resolution
