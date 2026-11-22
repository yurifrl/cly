-- Resume a plain claude TUI after the session daemon restarts.
--
-- Tern reopens `tern acp` and omp chats itself; a claude started with
-- `c --cli` is just a program in a shell pane, so a daemon restart (crash,
-- update it can't take in place) leaves a bare shell. `cly claude-tern-pane`
-- (a Claude SessionStart/SessionEnd hook) keeps one record per pane running
-- such a claude in ~/.local/state/cly/tern-claude/<TERN_PANE>.json. When the
-- daemon spawns that pane's shell again (same pane id), this hook has fish
-- run `cly claude --cli [flags] --resume <session>` at its first prompt, or
-- the original `cly claude --cli [flags] [args]` when the session was never
-- saved. The line is typed and executed at the prompt (not run by `-C`) so
-- Tern sees the command, names the tab after it and shows its program icon;
-- it starts with `c` when fish has that function (the usual alias).

local DIR = (tern.getenv("HOME") or "") .. "/.local/state/cly/tern-claude/"

-- fish single quotes: only \ and ' need escaping.
local function quote(s)
    return "'" .. string.gsub(string.gsub(s, "\\", "\\\\"), "'", "\\'") .. "'"
end

local function record(pane)
    local path = DIR .. pane .. ".json"
    if not tern.fs.exists(path) then return nil end
    local ok, rec = pcall(function() return tern.json.decode(tern.fs.read(path, 65536)) end)
    if not ok or type(rec) ~= "table" then return nil end
    if type(rec.session_id) ~= "string" or not string.match(rec.session_id, "^[%w%-]+$") then return nil end
    return rec
end

tern.on("spawn", function(spec)
    local pane = spec.env.TERN_PANE
    if not pane or not string.match(pane, "^%d+$") then return spec end
    -- `-C` (init command, then interactive) is fish's; other shells keep a plain prompt.
    if string.match(spec.program, "[^/]+$") ~= "fish" then return spec end
    local rec = record(pane)
    if not rec then return spec end

    local words = {}
    local flags = type(rec.flags) == "table" and rec.flags or { "--cli" }
    for _, f in ipairs(flags) do
        if type(f) == "string" and string.match(f, "^%-%-[%w%-]+$") then table.insert(words, f) end
    end
    -- claude writes a session's transcript at its first message; without one
    -- (nothing sent yet, e.g. a fresh `--fork-session`) `--resume <id>` finds
    -- no conversation, so the original claude args are replayed instead.
    local resumable = type(rec.transcript_path) ~= "string" or rec.transcript_path == ""
        or tern.fs.exists(rec.transcript_path)
    if resumable then
        table.insert(words, "--resume")
        table.insert(words, rec.session_id)
    elseif type(rec.args) == "table" then
        for _, a in ipairs(rec.args) do
            if type(a) == "string" then table.insert(words, quote(a)) end
        end
    end
    local rest = table.concat(words, " ")
    local init = ""
    if type(rec.cwd) == "string" and rec.cwd ~= "" and tern.fs.exists(rec.cwd) then
        init = "cd " .. quote(rec.cwd) .. "; "
    end
    init = init
        .. "function __cly_tern_restore --on-event fish_prompt; functions -e __cly_tern_restore; "
        .. "if functions -q c; commandline -r " .. quote("c " .. rest) .. "; "
        .. "else; commandline -r " .. quote("cly claude " .. rest) .. "; end; "
        .. "commandline -f execute; end"

    local args = {}
    for _, a in ipairs(spec.args) do table.insert(args, a) end
    table.insert(args, "-C")
    table.insert(args, init)
    spec.args = args
    return spec
end)
