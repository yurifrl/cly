#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
MODE="${1:-all}"
REFERENCE="$ROOT/.agents/tests/zs-smart-sessionizer/fixtures/reference-zellij-smart-sessionizer.sh"

build() {
  echo "=== Build cly ==="
  (cd "$ROOT" && GOFLAGS= go build -o dist/cly .)
  echo "✅ Build succeeded"
  echo
}

compare_mode() {
  local mode="$1"
  local workdir
  workdir="$(mktemp -d)"
  trap 'rm -rf "$workdir"' RETURN

  mkdir -p "$workdir/bin" "$workdir/layouts"
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
printf '%s\n' "\$*" >> "\${ZS_LOG_DIR}/zellij.log"
exit 0
EOF

  cat > "$workdir/bin/fzf" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
count_file="${ZS_LOG_DIR}/picker-count"
count=0
if [[ -f "$count_file" ]]; then
  count="$(cat "$count_file")"
fi
count=$((count + 1))
printf '%s' "$count" > "$count_file"
input="${ZS_LOG_DIR}/picker-${count}.txt"
cat > "$input"

mode="${ZS_MODE}"
case "$mode" in
  outside)
    if [[ "$count" -eq 1 ]]; then
      sed -n '3p' "$input"
    else
      tail -1 "$input"
    fi
    ;;
  attach)
    sed -n '1p' "$input"
    ;;
  inside)
    if [[ "$count" -eq 1 ]]; then
      sed -n '1p' "$input"
    else
      tail -1 "$input"
    fi
    ;;
esac
EOF
  cp "$workdir/bin/fzf" "$workdir/bin/sk"

  chmod +x "$workdir/bin/zoxide" "$workdir/bin/zellij" "$workdir/bin/fzf" "$workdir/bin/sk"

  run_target() {
    local target="$1"
    local dir="$workdir/$target"
    mkdir -p "$dir"
    local -a env=("PATH=$workdir/bin:$PATH" "ZS_LOG_DIR=$dir" "ZS_MODE=$mode")
    if [[ "$mode" == "inside" ]]; then
      env+=("ZELLIJ=1")
    fi

    if [[ "$target" == "reference" ]]; then
      env -i "${env[@]}" HOME="$HOME" bash "$REFERENCE" >/dev/null 2>&1
    else
      env -i "${env[@]}" HOME="$HOME" "$ROOT/dist/cly" zs >/dev/null 2>&1
    fi
  }

  echo "=== Compare $mode flow ==="
  run_target reference
  run_target ours

  local files=(picker-1.txt zellij.log)
  if [[ "$mode" != "attach" ]]; then
    files+=(picker-2.txt)
  fi

  for file in "${files[@]}"; do
    echo "--- $file"
    diff -u "$workdir/reference/$file" "$workdir/ours/$file"
  done

  echo "✅ PASS: $mode flow matches reference"
  echo
}

build

case "$MODE" in
  outside)
    compare_mode outside
    ;;
  attach)
    compare_mode attach
    ;;
  inside)
    compare_mode inside
    ;;
  all)
    compare_mode outside
    compare_mode attach
    compare_mode inside
    ;;
  *)
    echo "usage: $0 [outside|attach|inside|all]" >&2
    exit 1
    ;;
esac
