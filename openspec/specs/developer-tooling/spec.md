# developer-tooling Specification

## Purpose
TBD - created by archiving change add-documentation-and-tooling. Update Purpose after archive.
## Requirements
### Requirement: Module Template Documentation
The project SHALL provide comprehensive module template documentation in docs/module-template.md.

#### Scenario: Template reflects current architecture
- **WHEN** developer reads module-template.md
- **THEN** it SHALL document demo namespace pattern (modules/demo/)
- **AND** document utility module pattern (modules/<utility>/)
- **AND** reflect current file structure with cmd.go + implementation

#### Scenario: Reference examples are current
- **WHEN** template lists reference examples
- **THEN** it SHALL mention 48 Bubbletea examples in references/
- **AND** provide mapping of commands to reference examples
- **AND** explain how to adapt reference code to module structure

#### Scenario: File templates match codebase
- **WHEN** developer uses template code
- **THEN** cmd.go template SHALL match pattern in existing modules
- **AND** implementation template SHALL include initialModel() function
- **AND** templates SHALL compile without modifications

### Requirement: Claude Code Skill for Module Creation
The project SHALL provide a Claude Code skill for automated module scaffolding.

#### Scenario: Skill file exists and is discoverable
- **WHEN** Claude Code is used in the project
- **THEN** skill SHALL exist at .claude/skills/add-module/SKILL.md
- **AND** skill SHALL have proper YAML frontmatter with name and description
- **AND** Claude SHALL automatically discover skill when module creation is discussed

#### Scenario: Skill description is specific
- **WHEN** skill frontmatter is read
- **THEN** description SHALL specify it creates demo modules
- **AND** mention it follows Bubbletea/Bubbles conventions
- **AND** mention Cobra CLI integration

#### Scenario: Skill provides step-by-step instructions
- **WHEN** skill is invoked
- **THEN** it SHALL provide 5-step workflow (identify, create dir, cmd.go, implementation, test)
- **AND** include code templates for cmd.go and implementation
- **AND** reference existing modules as examples

#### Scenario: Skill includes templates
- **WHEN** developer uses skill to create module
- **THEN** skill SHALL provide cmd.go template with Register() function
- **AND** provide implementation template with initialModel()
- **AND** templates SHALL use correct package names and imports

#### Scenario: Skill documents validation
- **WHEN** module is scaffolded
- **THEN** skill SHALL include validation checklist
- **AND** checklist SHALL include compilation test
- **AND** checklist SHALL include registration verification
- **AND** checklist SHALL include runtime test

### Requirement: Module Template Tea Options
The template documentation SHALL explain when and how to use Bubbletea program options.

#### Scenario: Common options documented
- **WHEN** developer needs special Bubbletea features
- **THEN** template SHALL document tea.WithAltScreen() usage
- **AND** document tea.WithMouseAllMotion() usage
- **AND** document tea.WithReportFocus() usage
- **AND** provide examples of when each option is needed

#### Scenario: Options are shown in context
- **WHEN** viewing option documentation
- **THEN** each option SHALL reference a demo that uses it
- **AND** show exact usage in cmd.go RunE function

### Requirement: Module Categories
The documentation SHALL distinguish between demo and utility modules.

#### Scenario: Demo modules defined
- **WHEN** creating demonstration module
- **THEN** documentation SHALL specify it goes in modules/demo/<name>/
- **AND** explain demos showcase UI components
- **AND** provide examples: spinner, chat, table

#### Scenario: Utility modules defined
- **WHEN** creating utility module
- **THEN** documentation SHALL specify it goes in modules/<name>/
- **AND** explain utilities provide real functionality
- **AND** provide examples: uuid generator

#### Scenario: Selection guidance provided
- **WHEN** deciding module category
- **THEN** documentation SHALL explain when to use demo namespace
- **AND** explain when to create top-level utility
- **AND** provide decision criteria

