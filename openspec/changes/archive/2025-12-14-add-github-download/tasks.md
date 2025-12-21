# Tasks: Add GitHub Release Download

- [x] Add `zellij_plugin` function to parse GitHub URL and extract owner/repo
- [x] Implement HTTP download with redirect following
- [x] Create destination directory if needed
- [x] Make destination configurable via `modules.dotfiles.zellij_plugins_dir`
- [x] Write tests for URL parsing
- [x] Write integration test with real download (skipped in CI)
- [x] Update executeCommand to call new implementation
