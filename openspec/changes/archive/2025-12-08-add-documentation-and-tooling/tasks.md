# Implementation Tasks

## 1. README Creation
- [x] 1.1 Create README.md with project description
- [x] 1.2 Add installation instructions (go install, build from source)
- [x] 1.3 Add usage examples (uuid, demo commands)
- [x] 1.4 Create commands table listing all utilities
- [x] 1.5 Add architecture overview with module structure
- [x] 1.6 Document dependencies (Cobra, Bubbletea, Bubbles, Lipgloss)
- [x] 1.7 Add contributing section referencing module-template.md
- [x] 1.8 Add license section (MIT)

## 2. Module Template Update
- [x] 2.1 Update docs/module-template.md to reflect current structure
- [x] 2.2 Document demo namespace pattern (modules/demo/)
- [x] 2.3 Update reference examples (48 demos available)
- [x] 2.4 Add section on demo vs utility modules
- [x] 2.5 Update file templates to match current code patterns
- [x] 2.6 Add section on using tea.WithAltScreen() and other options

## 3. Claude Code Skill Creation
- [x] 3.1 Create .claude/skills/add-module/ directory
- [x] 3.2 Write SKILL.md with frontmatter (name, description)
- [x] 3.3 Add skill instructions for scaffolding modules
- [x] 3.4 Include cmd.go template in skill
- [x] 3.5 Include implementation file template
- [x] 3.6 Document pattern: extract from references/bubbletea/examples/
- [x] 3.7 Add validation checklist (compiles, registers, runs)
- [x] 3.8 Document when to use demo vs utility modules

## 4. Documentation Polish
- [x] 4.1 Ensure README reflects actual commands (uuid + demo)
- [x] 4.2 Update command count (48 demos)
- [x] 4.3 Add examples for top 5 most useful demos
- [x] 4.4 Verify all links and references are correct

## 5. Validation
- [x] 5.1 Verify README.md renders correctly on GitHub
- [x] 5.2 Test skill is discoverable (restart Claude session)
- [x] 5.3 Verify module-template.md instructions are accurate
- [x] 5.4 Run through "adding a module" workflow end-to-end

## Dependencies
- Section 2 can run in parallel with section 1
- Section 3 can run in parallel with sections 1 and 2
- Section 4 depends on sections 1, 2, 3 completion
- Section 5 depends on section 4 completion

## Parallel Work
- Tasks 1.1-1.8 can be written simultaneously (same file)
- Tasks 2.1-2.6 can be edited in parallel (same file)
- Tasks 3.1-3.8 can be written simultaneously (SKILL.md)
