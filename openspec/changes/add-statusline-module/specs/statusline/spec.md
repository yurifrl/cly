## ADDED Requirements

### Requirement: Main Command
The system SHALL provide `cly statusline` that outputs composed line from config.

#### Scenario: Config-driven output
- **WHEN** config has context.enabled=true, model.enabled=true
- **THEN** output "🧠 45% (90K/200K) │ [Opus]"

#### Scenario: Respect disabled
- **WHEN** config has model.enabled=false
- **THEN** exclude model from output

### Requirement: Context Subcommand
The system SHALL provide `cly statusline context`.

#### Scenario: Output context
- **WHEN** stdin has context_window
- **THEN** output "🧠 45% (90K/200K)"

#### Scenario: Color by threshold
- **WHEN** percentage >= config.context.danger (default 75)
- **THEN** red with 🔴

#### Scenario: Custom symbol
- **WHEN** config.context.symbol = "📊"
- **THEN** use that symbol

### Requirement: Model Subcommand
The system SHALL provide `cly statusline model`.

#### Scenario: Output model
- **WHEN** stdin has model.display_name "Opus"
- **THEN** output "[Opus]"

### Requirement: Cost Subcommand
The system SHALL provide `cly statusline cost`.

#### Scenario: Output cost
- **WHEN** stdin has cost.total_cost_usd 0.02
- **THEN** output "💰 $0.02"

### Requirement: Format String
The system SHALL use format string to control output order.

#### Scenario: Parse format
- **WHEN** config has format = "$context │ $model"
- **THEN** output components in that order with separator

#### Scenario: Skip disabled
- **WHEN** format has $model but model.enabled=false
- **THEN** exclude $model from output

### Requirement: Custom Command
The system SHALL support custom command when config.custom.enabled=true.

#### Scenario: Execute custom command
- **WHEN** custom.enabled=true and custom.command set
- **THEN** execute command, include output

#### Scenario: Timeout protection
- **WHEN** custom.timeout=500 and command takes longer
- **THEN** kill command, skip output

#### Scenario: Starship via custom
- **WHEN** custom.command = "cd $cwd && starship prompt"
- **THEN** execute starship, include output

### Requirement: Config Options
Each component SHALL support enabled flag and params.

#### Scenario: Component config
- **WHEN** config has context.enabled, context.symbol, context.warning
- **THEN** apply those params

### Requirement: Integration Test
The system SHALL have integration test with JSON input/output.

#### Scenario: Full flow
- **GIVEN** sample StatusJSON
- **WHEN** piped to `cly statusline`
- **THEN** output matches expected format
