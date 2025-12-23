# CLY Documentation System Specification

**Purpose:** Define what documentation exists, where it lives, what it's for.

---

## Documentation Types

### 1. AI Navigation Layer

**What:** Help AI find code, understand context, make decisions
**Where:** TBD (`.ai/`? `docs/ai/`? inline comments?)
**Format:** Markdown with YAML frontmatter
**Length:** < 500 words per doc

**Should contain:**
- Module purposes (what each module does)
- Architecture decisions (WHY choices made)
- Pattern library (how to use Bubbletea, Cobra, Store, Config)
- Triggers (keywords that route to relevant docs)
- Antipatterns (what NOT to do)

**Should NOT contain:**
- Code tutorials (AI doesn't follow steps)
- API reference (code has that)
- What code clearly shows (focus on WHY and WHEN)

### 2. Blueprints

**What:** Rough feature specs to pass to OpenSpec
**Where:** TBD (`.ai/blueprints/`? `blueprints/`? `docs/blueprints/`?)
**Format:** Markdown, opinionated
**Length:** Whatever needed (100-600 lines)

**Should contain:**
- Opinionated architecture (HOW to build)
- Implementation approach
- Trade-offs and decisions
- Open questions

**Relationship to OpenSpec:**
- Blueprint = rough idea with HOW
- OpenSpec = formalized requirements (WHAT)
- Workflow: Write blueprint → AI converts to OpenSpec proposal

### 3. Code Documentation

**What:** Inline docs for understanding code
**Where:** In the code (GoDoc comments, inline comments)
**Format:** Go comments

**Should contain:**
- WHY code does something (not what)
- Edge cases, gotchas
- References to related code

**Should NOT contain:**
- Obvious descriptions ("this function adds two numbers")
- Duplicate information from type signatures

### 4. Human Docs

**What:** README, architecture, contributing
**Where:** `docs/`, `README.md`, module READMEs
**Format:** Markdown

**Should contain:**
- Project overview
- Installation
- Architecture overview (visual)
- Contributing guide

---

## Open Questions

**Where should AI navigation docs live?**
- Option A: `.ai/docs/` (synced to tools)
- Option B: `docs/ai/` (closer to other docs)
- Option C: Inline in code (closest to origin)
- Option D: Mix (decisions in docs/, patterns inline)

**Where should blueprints live?**
- Option A: `.ai/blueprints/` (synced to tools)
- Option B: `blueprints/` (existing location)
- Option C: `docs/blueprints/`

**What exactly goes in each doc type?**
- Module docs: Purpose? Patterns? Entry points? Examples?
- Architecture docs: All WHY decisions? Just major ones?
- Pattern docs: Full examples? Just structure?

**How to keep docs current?**
- Manual updates during release?
- Auto-detection with suggestions?
- Leave it to discipline?

---

## Principles (From Research)

1. **Signal over noise** - Only document what code can't show
2. **Close to origin** - Docs near code they describe
3. **Intent over implementation** - WHY and WHEN, not WHAT
4. **Scannable** - < 8 second skim to find info
5. **AI-first** - Optimize for AI navigation, not human reading
6. **Triggers** - Metadata so AI routes to right context
7. **No duplication** - One source of truth per concept

---

## What Needs Deciding

Before implementing anything, decide:

1. **Location strategy** - Where does each doc type live?
2. **Content strategy** - What exactly goes in each doc?
3. **Maintenance strategy** - How to keep current?
4. **Sync strategy** - How to get to AI tools?

---

**This is the definition. Implementation happens after these decisions are made.**
