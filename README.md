# hyprls — Lua type definitions for Hyprland configs

Hyprland [migrated its configuration language to Lua](https://hypr.land/news/26_lua/).
This repository ships a [lua-language-server][lua_ls] addon that gives you full
autocomplete, hover docs, and type checking inside `~/.config/hypr/hyprland.lua`.

It used to be a Go LSP for the legacy `hyprlang` config format; that work has
been retired in favour of leaning on `lua_ls`, which is already installed in
most editor setups.

[lua_ls]: https://luals.github.io/

## What's in here

```
library/
  hl.meta.lua          canonical hl.* API stub. Vendored from Hyprland upstream
                       (meta/generateLuaStubs.py); regenerate with `just gen-api`.
  hl.config.meta.lua   typed schema for the table accepted by hl.config({...}),
                       generated from the Hyprland wiki by `cmd/gen-lua`.
examples/
  .luarc.json          drop-in lua_ls config that loads the library
  hyprland.lua         sample config that exercises the types
cmd/gen-lua/           regenerator for hl.config.meta.lua
parser/data/           wiki ingestion (used only by the generator)
scripts/check-lua.sh   smoke-test the library with `lua-language-server --check`
```

## Tooling

- **Format / lint**: [stylua](https://github.com/JohnnyMorganz/StyLua),
  configured via `.stylua.toml` (2-space, `AutoPreferSingle`, no call parens)
  to match the rest of my Lua codebases. Run `just fmt` / `just lint`.
- **"Tests"**: there's no runtime Lua to test — the library is a pile of
  `---@meta` annotations. `just check` (or `./scripts/check-lua.sh`) runs
  `lua-language-server --check` against `examples/hyprland.lua`, which
  exercises every public surface and fails CI if anything regresses.
- **Go**: `go test ./...` covers the wiki ingestion that feeds the generator.

No `busted` / `luacheck` / `selene` — they don't add value for an
annotation-only library and would just slow CI down.

## Usage

1. Clone this repository somewhere stable, e.g. `~/.local/share/hyprls`.
2. Put a `.luarc.json` next to your `hyprland.lua` that points lua_ls at the
   library:

   ```json
   {
     "runtime.version": "Lua 5.4",
     "diagnostics.globals": ["hl"],
     "workspace.library": ["/home/you/.local/share/hyprls/library"],
     "workspace.checkThirdParty": false
   }
   ```

   See `examples/.luarc.json` for a starting template.
3. Open `hyprland.lua` — `lua_ls` will load the annotations and you'll get
   completion on `hl.`, on every option of `hl.config({ … })`, on dispatchers
   like `hl.dsp.window.move{ … }`, etc.

### Neovim (lspconfig) example

```lua
require("lspconfig").lua_ls.setup({
  settings = {
    Lua = {
      runtime = { version = "Lua 5.4" },
      diagnostics = { globals = { "hl" } },
      workspace = {
        library = { vim.fn.expand("~/.local/share/hyprls/library") },
        checkThirdParty = false,
      },
    },
  },
})
```

## Regenerating the stubs

Two stubs, two sources of truth:

- `library/hl.meta.lua` is produced by **Hyprland upstream's** own
  `meta/generateLuaStubs.py` (parses the C++ source). To refresh it against
  a local Hyprland checkout:

  ```sh
  just gen-api ~/src/Hyprland   # path to a hyprwm/Hyprland checkout
  ```

- `library/hl.config.meta.lua` augments the upstream stub with a typed
  schema for every option of `hl.config({...})`. It's derived from the
  bundled [hyprwm/hyprland-wiki](https://github.com/hyprwm/hyprland-wiki)
  submodule:

  ```sh
  just update     # pulls latest wiki + regenerates + stylua
  # or
  go run ./cmd/gen-lua library/hl.config.meta.lua && just fmt
  ```

## License

GPL-3.0 — see `LICENSE`.
