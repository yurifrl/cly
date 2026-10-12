---
name: "vectorcode"
description: "Semantic code search using RAG and vector embeddings. Use when you need to find code by concept/functionality rather than exact text match, discover implementation examples, or understand where specific concepts are used across projects."
allowed-tools:
  - mcp__vectorcode__ls
  - mcp__vectorcode__query
  - mcp__vectorcode__vectorise
  - mcp__vectorcode__files_ls
  - mcp__vectorcode__files_rm
---

# VectorCode - Semantic Code Search

VectorCode provides semantic code search using RAG (Retrieval-Augmented Generation) and vector embeddings. It indexes code files into ChromaDB and enables finding relevant code by meaning rather than exact text matching.

## When to Use This Skill

- Finding code with similar functionality but different naming conventions
- Discovering implementation examples across large codebases
- Understanding where specific concepts/patterns are used
- Retrieving contextual code for AI-assisted development
- Searching when you don't know exact variable/function names

## Available MCP Tools

### List Projects
**`mcp__vectorcode__ls`** - Get list of indexed projects (no parameters)

### Semantic Search
**`mcp__vectorcode__query`** - Search for code by concept
- `query_messages`: Array of keywords (separate phrases, include related terms)
- `n_query`: Number of files to retrieve (increase if context insufficient)
- `project_root`: Project path (get from `ls` first)

### Index Files
**`mcp__vectorcode__vectorise`** - Add files to index
- `paths`: Array of file paths (accurate, case-sensitive)
- `project_root`: Project identifier

### Manage Index
**`mcp__vectorcode__files_ls`** - List indexed files in project
**`mcp__vectorcode__files_rm`** - Remove files from index

## Query Best Practices

1. **Always run `ls` first** to get valid `project_root` values
2. **Break queries into keywords** - "auth login password" not "authentication login with password"
3. **Include related terms** - For "function" add: "return value", "parameter", "arguments"
4. **Add imported names** - If searching for imported class/function, include its name
5. **Try orthogonal keywords on retry** - If first query fails, use different but related terms
6. **Don't repeat exact keywords** - Avoid using same query terms from previous attempts
7. **Increase file count** - If results lack context, increase `n_query` parameter
8. **Don't escape special characters** - Use raw strings

## CLI Commands

Initialize project:
```bash
cd /path/to/project
vectorcode init                    # Setup project
vectorcode vectorise src/          # Index source directory
```

Update after changes:
```bash
vectorcode update                  # Re-index changed files
```

Search manually:
```bash
vectorcode query "authentication logic" -n 10
```

## Configuration

**Location**: `home/.config/vectorcode/config.json5`
```json5
{
    "db_url": "http://127.0.0.1:8000"  // ChromaDB server URL
}
```

**MCP Setup** (in `home/.claude/settings.json`):
```json
"vectorcode": {
    "command": "vectorcode-mcp-server"
}
```

## Result Handling

- **Provide references** - Include file paths and line ranges when answering
- **Don't paste full source** - Reference code locations, not entire files
- **Paths are relative** - All paths relative to project root
- **No external edits** - Don't suggest edits outside working directory

## Workflow Example

```
1. Run mcp__vectorcode__ls → get project list
2. Run mcp__vectorcode__query with:
   - query_messages: ["authentication", "user", "login", "password", "validation"]
   - n_query: 10
   - project_root: "/Users/yuri/Workdir/WIP/myapp"
3. Analyze returned files and line ranges
4. Provide answer with file references: "Authentication is handled in src/auth/login.py:45-78"
```

## Limitations

- Requires ChromaDB server running at configured URL
- Retrieval may not be accurate for single file requests (retrieve multiple)
- Results depend on quality of embeddings and chunking
- Best for semantic search, not exact pattern matching (use grep for that)

## Troubleshooting

**No results**: Check project indexed (`files_ls`), try broader keywords, increase `n_query`
**Files not updating**: Run `vectorcode update` or re-vectorize specific files
**Project not found**: Run `ls` to verify project_root value, check if initialized

## Key Difference from grep/ripgrep

- **VectorCode**: Finds conceptually similar code (semantic understanding)
- **grep**: Finds exact text patterns (literal matching)

Use VectorCode when you know WHAT you're looking for but not WHERE or HOW it's named.
