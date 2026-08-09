#!/usr/bin/env bash
# Recompute and lock the flake `vendorHash` for the cly Go module.
#
# buildGoModule needs a fixed-output hash of the module deps that only nix can
# compute. This automates the "set fakeHash -> build -> read the real hash"
# dance: it forces a mismatch, builds, scrapes the `got:` hash nix prints, and
# writes it back into flake.nix. Run it whenever go.mod / go.sum change.
#
# Uses a local `nix` if present, otherwise falls back to the nixos/nix Docker
# image (only go.mod/go.sum affect the result, so a dirty tree is fine).
set -euo pipefail
cd "$(dirname "$0")/.."

FAKE="sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
sed -i.bak -E "s|vendorHash = \"sha256-[^\"]*\";|vendorHash = \"${FAKE}\";|" flake.nix
rm -f flake.nix.bak

# Flakes only see git-tracked files; make new files visible without committing.
git add -N flake.nix >/dev/null 2>&1 || true

run_nix() {
  if command -v nix >/dev/null 2>&1; then
    nix --extra-experimental-features "nix-command flakes" build .#cly 2>&1
  else
    docker run --rm -v "$PWD":/src -w /src nixos/nix:latest \
      nix --extra-experimental-features "nix-command flakes" build .#cly 2>&1
  fi
}

echo "Building to discover vendorHash (this fetches all Go modules once)..."
OUT="$(run_nix || true)"
HASH="$(printf '%s\n' "$OUT" | grep -oE 'got:[[:space:]]*sha256-[A-Za-z0-9+/=]+' | grep -oE 'sha256-[A-Za-z0-9+/=]+' | head -1)"

if [ -z "${HASH}" ]; then
  echo "ERROR: could not determine vendorHash. Build output:" >&2
  printf '%s\n' "${OUT}" >&2
  exit 1
fi

sed -i.bak -E "s|vendorHash = \"sha256-[^\"]*\";|vendorHash = \"${HASH}\";|" flake.nix
rm -f flake.nix.bak
echo "vendorHash locked: ${HASH}"
