#!/usr/bin/env bash
# Run lua-language-server in --check mode against examples/. The example
# hyprland.lua exercises every public API in the library, so a clean run is
# our smoke test. Requires lua-language-server >= 3.7.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

cp "$ROOT/examples/hyprland.lua" "$TMP/"
cat >"$TMP/.luarc.json" <<JSON
{
  "runtime.version": "Lua 5.4",
  "diagnostics.globals": ["hl"],
  "workspace.library": ["$ROOT/library"],
  "workspace.checkThirdParty": false,
  "diagnostics.disable": ["lowercase-global", "duplicate-set-field"]
}
JSON

lua-language-server --check "$TMP" --logpath "$TMP" --checklevel=Warning >/dev/null
if [ -s "$TMP/check.json" ]; then
  echo 'lua_ls reported issues:' >&2
  cat "$TMP/check.json" >&2
  exit 1
fi
echo 'lua_ls: no issues'
