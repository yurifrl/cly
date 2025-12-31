# Base Converter Interface

```python
class IDEConverter(ABC):
    ide_name: str           # "claude", "opencode", etc.
    local_dir(): str        # ".claude", ".opencode"
    global_dir(): Path      # ~/.claude, ~/.config/opencode
    map_subdir(name): str   # commands → command (OpenCode)
    map_agent_file(name): str  # AGENTS.md → CLAUDE.md
    map_config_file(name): str | None  # claude.json → settings.json
    translate_skill(content, filename): str  # Content transformations
```
