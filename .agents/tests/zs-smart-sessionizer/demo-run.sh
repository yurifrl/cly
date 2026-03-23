#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TARGET="${1:-ours}"
MODE="${2:-outside}"
REFERENCE="$ROOT/.agents/tests/zs-smart-sessionizer/fixtures/reference-zellij-smart-sessionizer.sh"
REAL_FZF="$(command -v fzf)"

if [[ -z "$REAL_FZF" ]]; then
  echo "fzf not found" >&2
  exit 1
fi

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT
mkdir -p "$workdir/bin" "$workdir/layouts" "$workdir/logs"

printf 'pane\n' > "$workdir/layouts/dev.kdl"
printf 'tab name="main"\n' > "$workdir/layouts/tabbed.kdl"

cat > "$workdir/bin/zoxide" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "query" && "${2:-}" == "-l" ]]; then
  printf '%s\n' "/repo/alpha" "/repo/beta" "/repo/gamma"
  exit 0
fi
exit 1
EOF

cat > "$workdir/bin/zellij" <<EOF
#!/usr/bin/env bash
set -euo pipefail
if [[ "\${1:-}" == "list-sessions" && "\${2:-}" == "-s" ]]; then
  printf '%s\n' "dev" "work"
  exit 0
fi
if [[ "\${1:-}" == "setup" && "\${2:-}" == "--check" ]]; then
  printf 'LAYOUT DIR: "%s"\n' "$workdir/layouts"
  exit 0
fi
printf '%s\n' "\$*" >> "$workdir/logs/zellij.log"
exit 0
EOF

cat > "$workdir/bin/fzf" <<EOF
#!/usr/bin/env bash
set -euo pipefail
count_file="$workdir/logs/picker-count"
count=0
if [[ -f "\$count_file" ]]; then
  count="\$(cat "\$count_file")"
fi
count=\$((count + 1))
printf '%s' "\$count" > "\$count_file"
input="$workdir/logs/picker-\$count.txt"
cat > "\$input"
exec "$REAL_FZF" "\$@" < "\$input"
EOF
cp "$workdir/bin/fzf" "$workdir/bin/sk"
chmod +x "$workdir/bin/zoxide" "$workdir/bin/zellij" "$workdir/bin/fzf" "$workdir/bin/sk"

export PATH="$workdir/bin:$PATH"
export HOME="$HOME"
export FZF_DEFAULT_OPTS='--height=12 --border --layout=reverse --info=inline'

if [[ "$MODE" == "inside" ]]; then
  export ZELLIJ=1
else
  unset ZELLIJ || true
fi

clear
printf 'Target: %s\nMode: %s\n\n' "$TARGET" "$MODE"
printf 'Fake sessions: dev, work\n'
printf 'Fake zoxide dirs: /repo/alpha, /repo/beta, /repo/gamma\n'
printf 'Layouts: dev.kdl, tabbed.kdl, default\n\n'
printf 'Tips:\n'
printf '  - In picker 1, type: alpha\n'
printf '  - In picker 2, type: default\n\n'
printf 'Press Enter to start...'
read -r _

if [[ "$TARGET" == "reference" ]]; then
  bash "$REFERENCE"
else
  "$ROOT/dist/cly" zs
fi

printf '\n\n=== picker-1.txt ===\n'
cat "$workdir/logs/picker-1.txt"
if [[ -f "$workdir/logs/picker-2.txt" ]]; then
  printf '\n=== picker-2.txt ===\n'
  cat "$workdir/logs/picker-2.txt"
fi
printf '\n=== zellij.log ===\n'
cat "$workdir/logs/zellij.log"

printf '\nDone. Press Enter to exit...'
read -r _
