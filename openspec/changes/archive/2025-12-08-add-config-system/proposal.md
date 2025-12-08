# Change: Add Configuration System

## Why
CLY needs a configuration system to allow users to customize behavior (themes, default options, module settings) without recompiling. Viper provides YAML config files, environment variables, and defaults. This enables per-user customization, machine-specific settings, and module-specific configuration sections following the modular architecture.

## What Changes
- Add Viper dependency for configuration management
- Create default `config/config.yaml` embedded in binary
- Implement `pkg/config/config.go` for config loading
- Support config hierarchy: env vars (CLY_*) → ~/.config/cly/config.yaml → config/config.yaml → ./config.yaml
- Add `config` subcommands: `init`, `show`, `set`, `get`
- Enable module-specific config sections (e.g., `modules.uuid.default_version`)

## Impact
- **Affected specs**:
  - Creates new capability `config-management`
  - Creates new capability `viper-integration`
  - Modifies `cli-foundation` to initialize config on startup
- **Affected code**:
  - New: `config/config.yaml` (default configuration)
  - New: `pkg/config/config.go` (Viper-based loader)
  - New: `modules/config/` (config subcommands: init, show, set, get)
  - Modified: `cmd/root.go` (add config initialization in PersistentPreRunE)
  - Modified: Example modules to read config (optional enhancement)
- **Dependencies**:
  - Already has: `github.com/spf13/viper`
- **Testing**: Config loads, overrides work, commands function correctly
