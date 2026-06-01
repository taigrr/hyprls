# Sync the bundled Hyprland wiki sources to the latest documented version
# and regenerate the lua_ls library.

# Pull the wiki submodule and copy the relevant docs into parser/data/sources.
pull-wiki:
	#!/usr/bin/env bash
	set -euxo pipefail
	git submodule update --init --recursive --remote
	cp hyprland-wiki/pages/Configuring/*.md parser/data/sources/ 2>/dev/null \
		|| cp hyprland-wiki/content/Configuring/*.md parser/data/sources/

# Regenerate library/hl.config.meta.lua (the typed schema for hl.config).
gen:
	go run ./cmd/gen-lua library/hl.config.meta.lua
	just fmt

# Regenerate library/hl.meta.lua from the canonical Hyprland upstream
# generator. Requires Python 3 and a checkout of hyprwm/Hyprland.
gen-api hyprland-src:
	python3 {{hyprland-src}}/meta/generateLuaStubs.py --root {{hyprland-src}} --output library/hl.meta.lua

# Pull wiki + regenerate the config schema.
update: pull-wiki gen

fmt:
	stylua examples library/hl.config.meta.lua

lint:
	stylua --check examples library/hl.config.meta.lua

# Smoke-test the library with lua-language-server against examples/.
check:
	./scripts/check-lua.sh

test:
	go test ./...
	just check
