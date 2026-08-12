# Design: Named AI provider list with conditions, weights, and default

Date: 2026-08-12
Status: Approved (design conversation)
Owner: pkg/ai

## Problem

`ai:` config today is a single global provider (`ai.provider`) plus a map of
per-provider blocks (`ai.providers.<type>`). Switching providers means editing
the file. There is no way to:

- define multiple OpenAI-compatible endpoints (url + token + model) as named entries,
- auto-pick a provider based on runtime context (user, host, arch, dir, env),
- declare a fallback default,
- see *why* a given provider was picked.

## Goals

1. `ai.providers` becomes a **list of named entries**; each entry describes one
   endpoint (provider type, base_url, api_key, model).
2. A new `condition` field per entry: a small expression evaluated against a
   runtime context (user, host, arch, os, dir, env.NAME).
3. A `weight` field (default 0) breaks condition ties; a `default: true` field
   marks the fallback; if no entry is default, the first entry is the fallback.
4. Selection always produces a **decision record**; verbose/debug mode prints
   the full context and per-entry evaluation.
5. Old map shape and `ai.provider` die. No backward-compat shim.

## Non-goals

- Module-level overrides (`modules.<x>.ai`) keep working exactly as today
  (layered on top of the picked entry). Their shape is unchanged.
- No new third-party dependency for the expression language in v1 — a tiny
  built-in evaluator. If conditions outgrow it, swap in expr-lang behind the
  same internal interface.

## Config shape (new)

```yaml
ai:
  providers:
    - name: aihub                      # required, unique
      provider: openai                 # anthropic | openai | openrouter | bedrock
      base_url: https://aihub-gateway.fbrai.dev/v1
      api_key: $AIHUB_API_KEY          # literal, $ENV, or ${ENV}
      model: aihub/claude-sonnet-5
      weight: 10                       # optional, default 0
      condition: 'user == "yuri" && dir =~ "~/Workdir/Yuri/*"'
      default: true                    # fallback when no condition matches

    - name: bedrock
      provider: bedrock
      model: us.anthropic.claude-sonnet-4-5-20250929-v1:0
      condition: 'user == "yuri" && dir =~ "~/Workdir/Nsx/*"'
```

Removed: `ai.provider`, map-shaped `ai.providers`.

## Selection algorithm

1. Build runtime context once per process:
   - `user` — os/user username
   - `host` — os.Hostname
   - `arch` — runtime.GOARCH
   - `os` — runtime.GOOS
   - `dir` — os.Getwd
   - `env.NAME` — env var value (empty string if unset)
2. Evaluate each entry's `condition` (missing condition = no match).
3. Among matches: highest `weight` wins; ties → earliest list position.
4. No match → entry with `default: true`; multiple defaults → first default;
   no default → first entry.
5. Apply the module-override layer on top of the picked entry, unchanged.

## Condition language

Built-in evaluator, no new dep. Grammar (recursive descent):

```
expr    := or
or      := and ("||" and)*
and     := unary ("&&" unary)*
unary   := "!" unary | "(" expr ")" | comparison
comparison := operand (("==" | "!=" | "=~") operand)?
operand := field | string_literal
field   := "user" | "host" | "arch" | "os" | "dir" | "env." NAME
```

- `==`, `!=` — string equality.
- `=~` — glob match on strings (`*` and `?` via path.Match semantics, but on
  the full string, not path segments). `dir =~ "~/Workdir/*"` style.
- Bare operand (e.g. `env.CLY_WORK`) is truthy when non-empty.
- `~` in patterns is expanded to the home dir before matching, and `dir` is
  compared in absolute form; a `~`-leading pattern matches the absolute path
  under home. (Implementation detail: expand pattern, not the value.)

Errors (fail fast at config load, not silently false):
- unknown field name
- syntax error (with position)
- duplicate entry `name`
- zero entries under `providers:`

## Decision record & logging

Selection returns `(entry, decision)` where decision captures:

- full context map used
- per-entry: name, condition string, match result, weight, chosen reason
- picked name + reason (`condition match`, `default`, `first entry`)

Logging behavior:
- **Always (when `app.debug` true):** one line to stderr:
  `ai: picked provider "aihub" (condition match, weight 10)`
- **Verbose dump:** with `app.debug` true, the full context table and each
  entry's evaluation printed to stderr.
- **`cly ai status`** (new command, `modules/ai/`): prints the same decision
  table without needing debug mode — for "why is it using THIS key?".
  Never prints resolved secret values; only env var names / "(set)"/"(unset)".

## Architecture

All new code lives in `pkg/ai/` (selection is core AI infra, not a module):

- `pkg/ai/providers.go` — entry struct, list parsing/validation from the
  `ai.providers` config block.
- `pkg/ai/context.go` — runtime context builder + decision record types.
- `pkg/ai/condition.go` — condition parser + evaluator (pure, table-tested).
- `pkg/ai/select.go` — selection algorithm; returns entry + decision.
- `pkg/ai/ai.go` — `resolve` switches from map-lookup to list-selection;
  `Resolved` struct and module-override layering unchanged.
- `modules/ai/cmd.go` — new `cly ai status` command printing the decision.

`Resolved.Provider` still feeds `llm.NewClient` — `pkg/llm` untouched.

Call sites (`ai.NewClient*`, `ai.HasAPIKey*`, `ai.Complete*`) keep their
signatures; behavior change is entirely inside resolution.

## Error handling

- Config-load errors (bad syntax, dup name, empty list, unknown field in
  condition): returned from selection with entry name + message. Callers
  surface them like today's client-creation errors.
- `HasAPIKey` on an error: returns false (UI gating stays safe).
- A condition that errors at eval time can't happen — all errors are caught
  at parse time (fields are validated against the known set).

## Testing

- `condition_test.go` — table-driven: operators, precedence, parens, glob,
  env lookup, `~` expansion, error cases.
- `select_test.go` — precedence: weight ordering, weight tie → list order,
  default fallback, first-entry fallback, multi-default → first default,
  module override on top of picked entry.
- Integration: `LoadConfig`/`LoadConfigWith` against fixture config maps.

## Migration

One-time manual edit of `~/.config/cly/config.yaml` (and repo default
`modules/config/config.yaml`) to the new list shape. Old shape produces a
clear load error telling the user the format changed.
