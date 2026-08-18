# Location-aware AI model failover

## Product outcome
CLY now keeps directory-aware model preference while automatically failing over to all configured models on any request error.

## Scope
Shared CLY AI client selection and git-commit planning requests.

## Evidence
- `go test ./pkg/llm ./pkg/ai ./modules/git-commits -count=1` passed.
- `pkg/llm/fallback.go` retries Complete and Stream through all configured candidates and aggregates final failures.
- `pkg/ai/ai.go` preserves location-based selection as first candidate then orders every configured provider as fallback.
- `modules/git-commits/pipeline.go` uses shared `ai.NewClientWith`, covering git-commits and git wip.
