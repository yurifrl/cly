# Documentation Plan - Questions for User

Based on comprehensive codebase research, I need your input on these strategic decisions:

## 1. Documentation Relationship: Blueprints vs OpenSpec

**Current state:**
- Blueprints = Tutorial-style implementation guides (how we built it)
- OpenSpec = Formal requirements + change tracking (what it should do)
- Some features have BOTH (aliexpress-scraper: 600-line blueprint + openspec proposal)

**Question:** Should I:
- **A)** Merge them (keep OpenSpec, archive blueprints as historical)
- **B)** Keep separate but define clear roles (blueprints = learning, openspec = spec)
- **C)** Something else?
leave as is

**Context:** Diátaxis framework suggests separating tutorials from reference, which matches your current split.
ok, never heard of it, it's interesting, but leave those there, those are ai resources

---

## 2. Documentation Format Preference

**Research found these patterns work well:**
- **ADRs** (Architecture Decision Records) - Lightweight decision documentation
- **YAML frontmatter** - Metadata for AI routing (triggers, applies_when, domain)
- **Diátaxis 4-pillar** - Tutorial, How-to, Reference, Explanation

**Question:** Which feels right for CLY?
- **A)** Add ADRs for major decisions (why Cobra? why zero-coupling?)
- **B)** Enhance existing docs with YAML metadata only
- **C)** Keep current structure, just fill gaps
- **D)** Hybrid approach?
I like hybrid

**My recommendation:** B + lightweight ADRs for critical decisions only

---

## 3. Demo Module Documentation

**Current:** 48 demo modules, 0 have READMEs

**Question:** Should demos get documentation?
- **A)** Yes - each demo gets minimal README (purpose, key patterns shown)
- **B)** No - demos are self-documenting code (reference implementations)
THis -> B
- **C)** Partial - only complex demos (chat, split-editors) get docs

**Context:** Demos already link to upstream Bubbletea examples. Adding docs might be redundant.

---

## 4. Documentation Priority

**Question:** What's more important right now?
- **A)** AI context (help Claude navigate and make good decisions)
this -> A
- **B)** Human onboarding (new contributors understand the project)
- **C)** Both equally


**This affects:** Length, tone, metadata richness, example density

---

## 5. Focus Area

**Gaps identified:**
1. Module-level context (40+ modules undocumented)
2. Architecture decisions (why choices made)
3. Stale docs cleanup (outdated blueprints, wrong paths)
4. Configuration guide (precedence unclear)
5. Pattern library (reusable how-tos)

**Question:** Top 3 priorities from this list?
1, 2, 5 (but all are good)

---

## 6. Documentation Living Location

**Current spread:**
- Global: `.claude/` (skills, agents)
- Project: `openspec/`, `blueprints/`, `docs/`
- Module: `modules/*/README.md` (mostly missing)

**Question:** Should I:
- **A)** Consolidate to one place (where?)
- **B)** Keep distributed but add index/map
- **C)** Current structure is fine, just fill gaps
Yes, theres no option here yet, talking about the ai things, there rest are workable

**Your preference:** "docs should live close to origin (code) but favor global ai framework"

---

## 7. Specific Patterns to Document

**From research, these need explanation:**
1. Register() pattern (why it enables zero-coupling)
2. Config precedence (local > system > embedded > env vars)
3. Store interface (when to use vs config)
4. Module isolation (portability test)
5. TDD philosophy (integration by default, <2 mocks)

**Question:** Any of these you DON'T want documented? Any missing?
All are good Id add maybe docs should serve like claude code skills, they should help ai to find things faster and eficiently without fillling context

---

## 8. Human Scannability Target

**Research:** Average attention span = 8 seconds, users skim not read

**Question:** Max acceptable length for:
- Module README: _____ (my suggestion: 1 A4 page ~500 words)
- Architecture doc: _____ (my suggestion: 3 pages ~1500 words)
- How-to guide: _____ (my suggestion: 1-2 pages ~750 words)
- Pattern reference: _____ (my suggestion: < 1 page ~400 words)
sounds good ur suggestions
---

## 9. Metadata Framework

**Question:** Should docs have structured metadata for AI routing?

Example:
```yaml
---
name: module-registration-pattern
triggers: ["Register()", "adding module", "new command"]
applies_when: ["Creating CLI commands", "Module structure"]
domain: architecture
level: beginner
antipatterns: ["cross-module imports", "global state"]
---
```

**Vote:** Yes / No / Only for skills, not general docs: Yes

---

## 10. Stale Documentation Handling

**Found stale:**
- `TODO.md` (5 orphaned items) leave it, its wip
- `docs/architecture.md` (references completed phases as future)
- Some blueprint paths wrong

**Question:** Should I:
- **A)** Archive stale docs to `docs/archive/`
- **B)** Update them to current state
- **C)** Delete entirely
- **D)** Mark as historical with disclaimer
do what u think is best, just dont delete

---

## Your Turn

Answer what you want, skip what you don't care about. I'll use your answers to build the documentation plan.

**Quick answers work:** Just "3B, 4A, 5: modules+arch+config, 9: yes" is fine.

Just focus on Ai, this is literaly a repo just for ME, no one will use but me, maybe think that if i get back to it  long time n the future, I might want to skim docs, and we live in a world we dont need to undesrtand things, we just ask ai, so removing noise is best, think on my workflow, not vibe coding perse but with litte to none iteraction with code
