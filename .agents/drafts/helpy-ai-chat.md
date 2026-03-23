# Helpy AI Chat

Embed an AI chat panel in the helpy TUI doc viewer so users can ask questions about the document they're reading.

> Note: This is a first draft to organize initial ideas before creating a formal OpenSpec proposal.

## Architecture

Replace the subprocess-based `mods` wrapper (`modules/llm-chat/`) with a native Go LLM package (`pkg/llm/`) using official Anthropic and OpenAI SDKs. Embed a chat sub-component in the helpy viewer that streams tokens directly into a bubbletea viewport via `tea.Cmd`/`tea.Msg`.

## Components

- **`pkg/llm/`** — Unified LLM client with `StreamReader` interface (`Next()`, `Text()`, `Err()`, `Close()`); Anthropic and OpenAI providers share identical streaming patterns
- **`modules/helpy/chat.go`** — Chat sub-component (textarea + viewport + spinner), follows `demo/chat/chat.go` layout
- **`modules/helpy/frontmatter.go`** — YAML frontmatter parser for doc metadata (`name`, `description`, `url`)
- **Config (`modules.helpy.ai`)** — Provider, model, API key (with `op://` 1Password support), system prompt

## Key Decisions

- **Direct SDK over mods subprocess** — eliminates process overhead, enables real-time token streaming into bubbletea, removes external binary dependency
- **Delete `modules/llm-chat/`** — fully replaced by `pkg/llm/`; no reason to keep the mods wrapper
- **Chat as toggle mode** — `ctrl+a` opens chat (bottom 40%), `esc` closes; viewport resizes dynamically
- **Doc content as system prompt context** — current document + frontmatter metadata passed to AI so it knows what you're reading
- **API key resolution**: config `op://` → config env var name → default provider env var

## Implementation Notes

- Go: `anthropics/anthropic-sdk-go`, `openai/openai-go` (both official, Go 1.22+)
- Reuse: `demo/chat` pattern, `bubbles/textarea`, `bubbles/viewport`, `glamour` (all already in project)
- Frontmatter: `gopkg.in/yaml.v3` (already in go.mod)
- Conversation history: in-memory slice per session, not persisted
