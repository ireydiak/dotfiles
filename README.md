# dotfiles

My macOS development setup, reproduced by a small Go tool called `dot`.
Configs live in `home/` and are symlinked into `$HOME`, so editing a live
config edits the repo. Packages live in the `Brewfile`.

## New Mac

```bash
xcode-select --install          # wait for it to finish
curl -fsSL https://raw.githubusercontent.com/ireydiak/dotfiles/main/bootstrap.sh | bash
```

Then work through the manual checklist that `dot install` prints:

1. Accessibility permissions for yabai and skhd (System Settings > Privacy & Security).
2. Partial SIP disable for yabai, from Recovery: `csrutil enable --without fs --without debug --without nvram`.
3. The yabai sudoers line (printed by `dot status`).
4. Restore SSH and GPG keys, then switch the remote to SSH:
   `git -C ~/.dotfiles remote set-url origin git@github.com:ireydiak/dotfiles.git`.
5. Create `~/.zshrc.local` with machine-specific exports. It is sourced by
   `.zshrc` and never tracked.

## Daily use

| Command | What it does |
|---|---|
| `dot` | Dashboard: packages, links, services, manual steps, repo state. Keys: `i` install, `l` link, `e` export, `u` update, `c` commit, `r` refresh, `q` quit |
| `dot status` | Same report as text; `--json` for machines |
| `dot install` | Trust taps, `brew bundle install`, link, run steps, start services. Safe to rerun |
| `dot link` | Create or repair symlinks. Existing files are moved to `~/.local/state/dot/backup/<timestamp>/` |
| `dot export` | Record newly installed brew packages into the Brewfile |
| `dot update` | `brew update`, `brew upgrade`, oh-my-zsh upgrade, nvim plugin sync |
| `dot commit` | `git add -A`, commit with a generated message, push |

Every command accepts `--dry-run`. Rebuild after pulling repo changes with
`go build -o ~/.local/bin/dot ./cmd/dot`.

## Machine-local files

Things that belong to one machine or one employer stay out of the repo:

- `~/.zshrc.local`: credentials and work aliases, sourced by the tracked `.zshrc`
- `~/.gitconfig.local`: included by the tracked `.gitconfig`, for example an `includeIf` that sets a work email under a work directory
- `~/.config/nvim/local-plugins/` plus a gitignored `lua/plugins/local-*.lua` spec: nvim plugins that stay on this machine

## Neovim

`~/.config/nvim` is the latest LazyVim starter, fetched by an install step, with
the repo's `lua/config`, `lua/plugins`, `snippets` and `lazyvim.json` symlinked
over it. Nothing pins plugin versions: `lazy-lock.json` is machine-local and
`dot update` runs `Lazy! sync`, so LazyVim and every plugin track upstream.
Breaking changes upstream are accepted as the price of staying current.

## Adding a config

1. Move the file or directory under `home/` at the path it has under `$HOME`.
2. Add one line to `[links]` in `manifest.toml`.
3. `dot link`, then `dot commit`.

## Layout

```
Brewfile        taps, formulae, casks (brew bundle format)
manifest.toml   links, install steps, launchd services, manual checklist
home/           mirrors $HOME
cmd/dot         the binary
internal/       exec, manifest, link, brew, steps, services, manual, repo, status, app, tui
```
