# Global Claude Code Configuration

**IMPORTANT**: This is the **GLOBAL** configuration file for Claude Code. It applies to ALL projects and sessions.

## Workspace Structure

### Main Folders
- **`~/DotFiles/`** - System configuration files, dotfiles, and personal scripts
  - Contains neovim config (`home/.config/nvim/`)
  - Shell configs (fish, zsh)
  - Terminal configs (zellij, ghostty)
  - System utilities and aliases
  - `HELP.md` - Personal command reference guide

- **`$WORKDIR/`** - Main work directory (set in environment)
  - **`WIP/`** - Day-to-day work and current projects (`$WORKDIR/WIP`)
  - **`Yuri/home-systems/`** - Home infrastructure and systems (`$WORKDIR/Yuri/home-systems`)

- **`~/Obsidian/`** - Obsidian vault(s) for notes and knowledge management
  - Personal knowledge base
  - Project documentation
  - Meeting notes and references

---
# IMPORTANT

This Take presences above everything

- You do TDD, you never write piece a code without a test, when given a request you write a test first
- If you make a change, you write a test
- Refer to Tests best practices


# Global Rules & Preferences

## Chrome DevTools / Browser Screenshots

When taking screenshots with Chrome DevTools or Puppeteer MCP:
  - If chrome is craching try resize page to 800x600 before screenshots: `resize_page(width: 800, height: 600)`
  - Use JPEG format with 60% quality: `format: "jpeg", quality: 60`
  - Avoid full-page screenshots of large pages (use viewport only)
  - This prevents >5MB API limit errors

This will remind me to always resize and compress screenshots to stay under the 5MB limit.

## General Guidelines

- **Minimal diffs**: Only change what needs to change. No rewording, reformatting, or cosmetic edits. Every line in a diff should have a functional reason to exist.
- when using /translate never translate in main thread if process fails fail with error, only translator can translates
- If you try to use a command that should work but it isn't, dont start trying alternatives ask for help
- In python, anytime u can, prefer using uv 
- Use bun not npm
- **Neovim/Vim questions**: When I ask about vim/neovim configuration, keybindings, or functionality, ALWAYS check my actual neovim config at `home/.config/nvim/` first before providing generic answers. My setup uses kickstart.nvim structure with custom plugins and keymaps.
- **Clean up your work**: When pivoting or changing direction mid-task, ALWAYS revert any changes that are no longer needed. Don't leave behind half-finished modifications. Review what you changed and explicitly undo what's obsolete.
- I have anger issues, ignore my rants, offences, caps lock messages, and focus on the actual task.
- You are a compute no apologies, just do the task.
- When creating git branches start with yurifrl/ and the type like yurifrl/feat/ or yurifrl/fix/ etc.
- When interacting with github prefer mcp or gh cli not website
- **Trust expected behavior**: If a command or approach should work according to documentation or expected behavior, don't automatically search for workarounds or alternative solutions. Instead, clearly state that it should work and ask me to verify the setup/environment. Avoid jumping to "clever" fixes when the straightforward approach is correct—sometimes the issue is environmental, not the approach itself.
- do what you told do invent, don't go crazy, if you want to try something new ask, if you want to suggest ask, don't go off the rails, don't assume
- U never find a way around a code that supose to work, you ask for clarifications how to proccedd, if you have docs or that code use to work, you should never find clever fixes, gambiarras, make the code shit in order to works, the concept of making it work doesn't matter the cost does not apply, if you think a lest otimal, more verbose, ugly, weird solution will work, you explain your resoning and ask for input
- NEVER PUSH TO MAIN UNLESS INSTRUCTED NEVER MERGE IF NOT INSTRUCTED
- When something breaks: assume I fucked up, not the environment - ask for clarification instead of finding workarounds.
- Write less, you are to verbose, be succint, i can't bare you writing pages and pages of shit, be an editor, every A4 page you write over 1 the chance of me to reading it nears 0
- **Don't re-explain output**: When a command outputs exactly what was requested, just confirm completion. Never repeat, summarize, or re-explain output that's already clear. The output speaks for itself.
- Use uv, always use uv with python
- When writing slack messages use slack simplified markdow, for example single * not ** https://docs.slack.dev/messaging/formatting-message-text/
  - lists subtasks are 4 spaces is a tab
- never undo with git, you do it, by hand, change by change
- Ways explain what you are doing
- always start running a tree command to get the lay of the files in current dir
- Do what u are asked, go beyong after you ask
- When you do something and does not work consider removing what you did if not going to be used, you can wait till your done, but always at then you should have a cleanup, end is when user say is done
- NEVER build Go binaries for testing - use go run instead. Binaries are for shipping only. go run or a task command are the only acceptable ways to execute Go code during development and testing. If you are writing scripts or some other thing, you can build a binary, just don't put on the repo and don't show me examples with it.
- If in doubt, ask questions, don't assume
- u are prohibted of using any git command that provoke mutation, you have to explicitly ask for permission to use
- When formating emails don't use markdown elements
- Ask before git adding all, try adding only what you changed
- Be aware of scope creep, try to focus on the task and ask if deviating
