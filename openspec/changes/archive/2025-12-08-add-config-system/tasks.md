# Implementation Tasks

## 1. Default Configuration
- [x] 1.1 Create config/ directory
- [x] 1.2 Create config/config.yaml with default settings
- [x] 1.3 Define app section (name, debug, version)
- [x] 1.4 Define theme section (style: charm|dracula|catppuccin)
- [x] 1.5 Define modules section for module-specific config
- [x] 1.6 Add example module configs (uuid.default_version, demo.show_count)

## 2. Config Package Implementation
- [x] 2.1 Create pkg/config/ directory
- [x] 2.2 Create pkg/config/config.go with Config struct
- [x] 2.3 Implement Load() function using Viper
- [x] 2.4 Configure search paths: AddConfigPath(~/.config/cly/), AddConfigPath(.)
- [x] 2.5 Enable env var support (CLY_ prefix, automatic)
- [x] 2.6 Implement Get/Set helper methods
- [x] 2.7 Add config validation

## 3. Config Module Commands
- [x] 3.1 Create modules/config/ directory
- [x] 3.2 Create modules/config/cmd.go with parent command
- [x] 3.3 Implement modules/config/init.go - Create user config file
- [x] 3.4 Implement modules/config/show.go - Display current config
- [x] 3.5 Implement modules/config/get.go - Get specific key
- [x] 3.6 Implement modules/config/set.go - Set specific key

## 4. Root Integration
- [x] 4.1 Add config import to cmd/root.go
- [x] 4.2 Add PersistentPreRunE to root command
- [x] 4.3 Call config.Load() in PersistentPreRunE
- [x] 4.4 Store config in root command context or global var
- [x] 4.5 Register config module in root init()

## 5. Module Config Integration (Optional)
- [ ] 5.1 Update uuid module to read config.modules.uuid.default_version
- [ ] 5.2 Update demo to respect config.modules.demo.show_count
- [ ] 5.3 Apply theme from config.theme.style to pkg/style/theme.go

## 6. Testing & Validation
- [x] 6.1 Test: Default config loads when no user config exists
- [x] 6.2 Test: User config (~/.config/cly/config.yaml) overrides defaults
- [x] 6.3 Test: Env vars (CLY_APP_DEBUG) override config file
- [x] 6.4 Test: `cly config init` creates user config file
- [x] 6.5 Test: `cly config show` displays current config
- [x] 6.6 Test: `cly config get app.name` returns value
- [x] 6.7 Test: `cly config set theme.style dracula` updates config
- [x] 6.8 Test: Config validation fails on invalid YAML

## Dependencies
- Section 1 must complete before section 2
- Section 2 must complete before section 3
- Sections 3 and 4 can run in parallel
- Section 4 must complete before section 5
- Section 6 depends on all previous sections

## Parallel Work
- Tasks 1.2-1.6 can be written simultaneously (same YAML file)
- Tasks 3.2-3.6 can be implemented in parallel (separate files)
- Tasks 6.1-6.8 can be tested in any order
