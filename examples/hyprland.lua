-- Minimal example Hyprland Lua config exercising the library types.
-- Drop this next to a `.luarc.json` (see examples/.luarc.json) that points
-- `workspace.library` at the `library/` directory of this repo.

hl.monitor { output = '', mode = 'preferred', position = 'auto', scale = 'auto' }

hl.config {
  general = {
    gaps_in = 5,
    gaps_out = 20,
    border_size = 2,
    col = {
      active_border = { colors = { 'rgba(33ccffee)', 'rgba(00ff99ee)' }, angle = 45 },
      inactive_border = 'rgba(595959aa)',
    },
    layout = 'dwindle',
  },
  decoration = {
    rounding = 10,
    blur = { enabled = true, size = 3, passes = 1 },
    shadow = { enabled = true, range = 4, color = 'rgba(1a1a1aee)' },
  },
  animations = { enabled = true },
}

hl.curve('easy', { type = 'spring', mass = 1, stiffness = 71.2633, dampening = 15.8273644 })
hl.animation { leaf = 'windows', enabled = true, speed = 4.79, spring = 'easy' }

local mainMod = 'SUPER'
hl.bind(mainMod .. '+Q', hl.dsp.exec_cmd 'kitty')
hl.bind(mainMod .. '+C', hl.dsp.window.close())
hl.bind(mainMod .. '+V', hl.dsp.window.float { action = 'toggle' })
hl.bind(mainMod .. '+left', hl.dsp.focus { direction = 'left' })
hl.bind(mainMod .. '+mouse:272', hl.dsp.window.drag(), { mouse = true })

for i = 1, 10 do
  local key = i % 10
  hl.bind(mainMod .. '+' .. key, hl.dsp.focus { workspace = i })
  hl.bind(mainMod .. '+SHIFT+' .. key, hl.dsp.window.move { workspace = i })
end

hl.window_rule {
  name = 'suppress-maximize',
  match = { class = '.*' },
  suppress_event = 'maximize',
}

hl.on('hyprland.start', function()
  hl.exec_cmd 'waybar &'
end)
