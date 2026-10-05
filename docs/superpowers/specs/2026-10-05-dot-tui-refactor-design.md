# `dot`: a small TUI to install and reproduce the macOS dev setup

Date: 2026-10-05
Status: approved design, awaiting implementation plan

## 1. Problem

The repo reproduces a developer setup on a new Mac by running bash scripts that
install packages and copy config files into place. It has become hard to
maintain because the live machine is the place where configs actually change,
and every change must be copied back into the repo by hand. In practice that
does not happen:

- 74 top-level Homebrew formulae and 12 casks are installed; the repo installs 10.
- nvim, tmux, skhd and yabai configs on the machine are newer than their repo
  copies. Ghostty, zsh, git and lazygit configs are not in the repo at all.
- The installer has bugs that would stop it on a fresh Mac (path doubling in the
  packaging loop, an Intel git DMG download, a misspelled skhd target directory,
  a hardcoded old username, rc files appended to on every run).
- `~/.zshrc` contains live API tokens, so it was never safe to add to the repo.

## 2. Decisions

These were made during the design conversation and are fixed for this version.

| Topic | Decision |
|---|---|
| Platform | macOS only (Apple Silicon, Homebrew at `/opt/homebrew`). Linux directory removed. |
| Config flow | Live paths are **symlinks into the repo**. Editing a live file edits the repo file. Capturing a change is `git commit`. No export step for configs. |
| Packages | Standard `Brewfile`, exported from the machine with `brew bundle dump`, installed with `brew bundle install`. |
| Tool | A single Go binary, `dot`, built with Bubble Tea. Every TUI action is also a plain subcommand. |
| Window manager stack | yabai, skhd, sketchybar are first-class managed items with their configs and services. |
| Themes | `themes/` directory and wallpapers removed. A future `dot theme <name>` command may return. |
| Manual steps | Tolerated where automation is impractical: Xcode CLT, SIP change for yabai, sudoers line, Accessibility permissions, SSH and GPG keys. The tool lists and detects them where it can. |
| Secrets | Never in the repo. `~/.zshrc` is split into a tracked file and an untracked `~/.zshrc.local`. |

## 3. Repository layout

```
dotfiles/
  Brewfile                      exported taps, formulae, casks
  manifest.toml                 what dot manages (links, steps, services, manual)
  bootstrap.sh                  fresh-machine entry point (bash, ~25 lines)
  README.md                     quickstart and manual checklist
  go.mod                        module github.com/ireydiak/dotfiles, Go 1.25
  cmd/dot/main.go               subcommand dispatch, flags, TUI entry
  internal/
    manifest/                   TOML schema, load, validate, tilde expansion
    repo/                       repo root resolution, git status/commit/push
    link/                       link planner and applier, backups
    brew/                       Brewfile parse, dump, diff, trust, install
    steps/                      idempotent step runner (check then run)
    services/                   launchd status and start
    manual/                     manual checklist with optional detection
    exec/                       command runner interface (real and fake)
    result/                     shared ok / skipped / failed result type and summary printing
    status/                     aggregates all of the above into one report
    app/                        wires the packages together; the CLI and TUI both call it
    tui/                        Bubble Tea dashboard and export checklist
  home/                         mirrors $HOME; every entry here is linked
    .zshrc
    .zprofile
    .gitconfig
    .config/git/ignore
    .config/tmux/tmux.conf
    .config/nvim/               full LazyVim config incl. init.lua, lazy-lock.json, lazyvim.json
    .config/skhd/skhdrc
    .config/yabai/{yabairc,create_spaces.sh}
    .config/sketchybar/         rc, items, plugins, helpers, helper/ sources (binary gitignored)
    Library/Application Support/com.mitchellh.ghostty/config
    Library/Application Support/lazygit/config.yml
  docs/superpowers/specs/       this document
```

Removed from the current repo: `linux/`, `macos/` (all scripts), `shared/`,
`themes/`, the vendored `sketchybar-app-font/` checkout, and the committed
`helper` Mach-O binary.

## 4. The manifest

`manifest.toml` at the repo root. Four sections. Values may use `~` for the
home directory; nothing else is expanded.

```toml
# repo path relative to home/  ->  absolute target under $HOME
[links]
".zshrc"                        = "~/.zshrc"
".zprofile"                     = "~/.zprofile"
".gitconfig"                    = "~/.gitconfig"
".config/git"                   = "~/.config/git"
".config/tmux"                  = "~/.config/tmux"
".config/nvim"                  = "~/.config/nvim"
".config/skhd"                  = "~/.config/skhd"
".config/yabai"                 = "~/.config/yabai"
".config/sketchybar"            = "~/.config/sketchybar"
"Library/Application Support/com.mitchellh.ghostty/config" = "~/Library/Application Support/com.mitchellh.ghostty/config"
"Library/Application Support/lazygit/config.yml"           = "~/Library/Application Support/lazygit/config.yml"

# non-brew installs; run in file order; `run` only executes when `check` exits non-zero
# `update` is optional and is only used by `dot update`
[[steps]]
id     = "oh-my-zsh"
check  = "test -d ~/.oh-my-zsh"
run    = "sh -c \"$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)\" '' --unattended --keep-zshrc"
update = "ZSH=~/.oh-my-zsh zsh ~/.oh-my-zsh/tools/upgrade.sh"

[[steps]]
id    = "nvm"
check = "test -s ~/.nvm/nvm.sh"
run   = "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | PROFILE=/dev/null bash"

[[steps]]
id    = "sketchybar-helper"
check = "test -x ~/.config/sketchybar/helper/helper"
run   = "make -C ~/.config/sketchybar/helper"

[[steps]]
id     = "nvim-plugins"
check  = "test -d ~/.local/share/nvim/lazy/lazy.nvim"
run    = "nvim --headless '+Lazy! restore' +qa"
update = "nvim --headless '+Lazy! sync' +qa"

# launchd agents; status is `launchctl print gui/<uid>/<label>` exit code
[[services]]
id    = "yabai"
label = "com.koekeishiya.yabai"
start = "yabai --start-service"

[[services]]
id    = "skhd"
label = "com.jackielii.skhd"
start = "skhd --install-service && skhd --start-service"

[[services]]
id    = "sketchybar"
label = "homebrew.mxcl.sketchybar"
start = "brew services start sketchybar"

# things only a human can do; `check` is optional
[[manual]]
id    = "xcode-clt"
check = "xcode-select -p"
how   = "xcode-select --install"

[[manual]]
id    = "sip-for-yabai"
check = "csrutil status | grep -q 'Filesystem Protections: disabled'"
how   = "Boot to Recovery, open Terminal, run: csrutil enable --without fs --without debug --without nvram"

[[manual]]
id    = "yabai-sudoers"
check = "sudo -n -l $(which yabai) --load-sa"
how   = "echo \"$(whoami) ALL=(root) NOPASSWD: sha256:$(shasum -a 256 $(which yabai) | cut -d' ' -f1) $(which yabai) --load-sa\" | sudo tee /private/etc/sudoers.d/yabai"

[[manual]]
id    = "accessibility"
how   = "System Settings > Privacy & Security > Accessibility: enable yabai and skhd"

[[manual]]
id    = "ssh-and-gpg"
how   = "Restore SSH and GPG keys, then: git -C ~/.dotfiles remote set-url origin git@github.com:ireydiak/dotfiles.git"
```

Validation rules, enforced on load:

- Every `[links]` key must exist under `home/`. Every value must start with `~/`.
- `id` values are unique within their section. `check` and `run`/`start`/`how`
  are non-empty strings. `update` on a step is optional; when present it is a
  non-empty string.
- Unknown keys are an error, so typos surface immediately.

Packages are intentionally not in the manifest. The `Brewfile` is Homebrew's
own declarative format and `brew bundle` already provides install, check and
dump. `dot` wraps it rather than duplicating it.

## 5. Brewfile handling and Homebrew 7 tap trust

Homebrew 7 refuses to load formulae from taps that have not been explicitly
trusted. On this machine that silently dropped `yabai` (tap `asmvik/formulae`)
and `sketchybar` (tap `felixkratz/formulae`) from `brew bundle dump`. The tool
therefore manages trust explicitly:

- **Install:** before `brew bundle install`, run `brew trust <tap>` for every
  `tap` line in the committed Brewfile. These taps are already committed, so no
  prompt.
- **Export:** read `brew tap`, and for any tap not yet trusted, show the list
  and ask for confirmation (`--yes` skips the prompt; the TUI shows a confirm
  dialog). Then run `brew bundle dump --force --file=<tmp>`.

`brew bundle dump` output is parsed into entries keyed by `kind` (`tap`,
`brew`, `cask`, `mas`, `vscode`) and `name`. Any trailing options on a line
(`link: true`, `restart_service: :changed`) are kept verbatim with the entry.

The export diff compares the dump against the committed Brewfile:

- **Added:** in the dump, not in the Brewfile. Default checked in the picker.
- **Removed:** in the Brewfile, not in the dump (uninstalled since). Default
  unchecked, so the Brewfile does not lose entries unless chosen.

The new Brewfile is written as the dump's lines filtered to
`committed ∪ chosen-added`, in dump order, followed by the committed entries
that are no longer installed and were not chosen for removal, in their
committed order. This keeps Homebrew's canonical ordering for everything
installed and makes diffs stable. `brew bundle cleanup` is never run
automatically.

## 6. Commands

`dot` is invoked as `dot <command> [flags]`. Without a command it opens the
TUI. Global flags: `--repo <path>`, `--dry-run`, `--yes`, `--json` (status
only). Exit code is 0 when every requested action succeeded or was a no-op,
1 when any action failed, 2 on usage or manifest errors.

| Command | Behavior |
|---|---|
| `dot status` | Prints one report: Brewfile entries not installed and installed packages not in the Brewfile, both from a temporary dump diffed against the committed Brewfile as in section 5 (no trust prompt, so untrusted-tap packages stay invisible until the first export; `brew bundle check` is not used because it also reports outdated packages); each link's state; each service loaded or not; each manual step done, pending or unverifiable; whether the repo has uncommitted changes. Never modifies anything. |
| `dot install` | In order: trust taps, `brew bundle install --file=Brewfile`, link everything, run each step whose check fails, start services that are not loaded. Continues past failures, prints a summary, exits 1 if any failed. Idempotent. Finishes by printing pending manual steps. |
| `dot link` | Applies the link plan only (see section 8). |
| `dot export` | Tap-trust prompt, dump, diff, picker (TUI) or print-and-confirm (CLI with `--yes` accepting the defaults), write Brewfile. |
| `dot update` | In order: `brew update`, `brew upgrade` (formulae and casks that are not self-updating), then the `update` command of every step that defines one, in manifest order. Installs nothing new and never runs `brew bundle cleanup`. Continues past failures, prints a summary, exits 1 if any failed. Ends by noting whether the repo now has uncommitted changes, since `Lazy! sync` rewrites `lazy-lock.json`. |
| `dot commit [-m msg] [--no-push]` | `git add -A`, commit with the given or a generated message, push if a remote exists and `--no-push` is absent. Generated message example: `dot: update .config/nvim, .config/tmux; Brewfile +3 -1`. |
| `dot` | The TUI dashboard (section 7). |

Linking runs before steps on purpose: the sketchybar helper and nvim plugin
steps operate on paths that must already point into the repo, and the
oh-my-zsh installer must see the tracked `.zshrc` (hence `--keep-zshrc`).

Repo root resolution, first match wins: `--repo` flag, `DOTFILES_DIR`
environment variable, the git top level of the current directory if it
contains `manifest.toml`, then `~/.dotfiles`. The resolved path is the symlink
target base. Moving the repo and running `dot link` repoints every link.

## 7. TUI

A single Bubble Tea program with two screens.

**Dashboard.** Four stacked sections matching `dot status`: Packages, Links,
Services, Manual. Each row is one item with a colored state glyph and a short
reason (for example `conflict: regular file exists`). A footer shows the
keybindings. Running an action opens a scrolling output pane at the bottom and
disables other keys until it finishes; the dashboard refreshes afterwards.

| Key | Action |
|---|---|
| `i` | install |
| `l` | link |
| `e` | export (opens the picker) |
| `u` | update |
| `c` | commit and push |
| `r` | refresh status |
| `q` | quit |

**Export picker.** A multi-select list with two groups, Added and Removed,
showing kind and name. Space toggles, `a` toggles all in a group, Enter writes
the Brewfile and returns to the dashboard, Esc cancels. Defaults as in
section 5. If the tap-trust prompt is needed it appears first as a yes/no
confirm.

The TUI contains no logic of its own. It calls the same functions the CLI
subcommands call and renders their results. Library choices: `bubbletea`,
`bubbles` (list, viewport), `lipgloss`. No cobra; subcommand dispatch uses the
standard `flag` package. TOML via `github.com/pelletier/go-toml/v2`.

## 8. Link engine

For each manifest link, with `src = <repo>/home/<key>` and `dst = <expanded value>`:

| Observed state of `dst` | Status | `dot link` action |
|---|---|---|
| symlink resolving to `src` | ok | none |
| does not exist | missing | `mkdir -p` parent, create symlink |
| symlink to another path | wrong-target | remove symlink, create new one; old target logged |
| regular file or directory | conflict | move to backup, create symlink |
| `src` missing in repo | broken-manifest | reported as error; no action |

"Resolving to `src`" means `os.Readlink(dst)`, made absolute and cleaned,
equals the absolute cleaned `src`. Links are always created with absolute
targets.

Backups go to `~/.local/state/dot/backup/<YYYYMMDD-HHMMSS>/<path relative to $HOME>`.
Nothing is ever deleted. With `--dry-run` the plan is printed and no filesystem
change happens. Each action is independent; one failure does not stop the rest.

## 9. Steps, services and manual checks

**Steps** run through `/bin/zsh -c` with `PATH` prefixed by `/opt/homebrew/bin`
and `HOME` set. `check` is run first; exit 0 means skip. Otherwise `run`
executes with output streamed, then `check` runs again. If it still fails the
step is marked failed. Steps run in manifest order so later steps can depend on
earlier ones.

A step's optional `update` command runs only from `dot update`, through the
same shell and environment, without consulting `check`. A non-zero exit marks
that step's update failed. Steps without `update` are skipped silently.

**Services** are loaded when `launchctl print gui/<uid>/<label>` exits 0.
`install` runs `start` only for services not loaded. `status` reports loaded or
not loaded; it does not restart anything.

**Manual** items with a `check` are reported done or pending from its exit
code. Items without a `check` are reported as "verify manually". The tool
stores no state about them.

All external commands go through an `exec.Runner` interface so tests inject
scripted outputs. Nothing in the test suite calls real `brew`, `launchctl` or
the network.

## 10. Secrets and machine-local configuration

The tracked `home/.zshrc` keeps everything shareable: oh-my-zsh setup, exports
of `EDITOR` and `GPG_TTY`, `PATH`, zoxide init, aliases and shell functions.
Its final lines are:

```zsh
# Machine-local settings and secrets. Not tracked.
[[ -f ~/.zshrc.local ]] && source ~/.zshrc.local
```

`~/.zshrc.local` is a plain file on each machine, created by the migration on
this one, holding the token exports (`NPM_TOKEN`, `GH_TOKEN`, `FLARE_API_KEY`
and the other credential variables currently in `.zshrc`). `dot` never reads,
writes or lists this file. The Flare-specific aliases and functions stay in the
tracked file; they contain no secrets and are wanted on a replacement work
machine.

The repo `.gitignore` also excludes `home/.config/sketchybar/helper/helper`
(built per machine) and the usual nvim scratch patterns already present in the
current nvim `.gitignore`.

## 11. Bootstrap on a fresh Mac

README quickstart:

```bash
xcode-select --install                      # manual, wait for it to finish
curl -fsSL https://raw.githubusercontent.com/ireydiak/dotfiles/main/bootstrap.sh | bash
```

`bootstrap.sh` does, in order, each step skipped if already satisfied:

1. Install Homebrew with the official installer and `eval "$(/opt/homebrew/bin/brew shellenv)"`.
2. `brew install go`.
3. `git clone https://github.com/ireydiak/dotfiles ~/.dotfiles` (HTTPS, no keys needed yet).
4. `mkdir -p ~/.local/bin && cd ~/.dotfiles && go build -o ~/.local/bin/dot ./cmd/dot`.
5. `~/.local/bin/dot install`.
6. Print the pending manual checklist and remind to open a new shell.

`~/.local/bin` is already on `PATH` in the tracked `.zprofile`/`.zshrc`.
Rebuilding `dot` itself after pulling repo changes is `go build` again;
`dot update` upgrades the managed software, not the `dot` binary or the repo.

## 12. Error handling

- Every unit of work (a brew install, a link, a step, a service start) returns
  a result of ok, skipped or failed with captured output. Runners never panic
  on a failing command.
- `install` and `link` continue past failures and end with a summary table and
  a non-zero exit when anything failed.
- Manifest or Brewfile parse errors abort immediately with file and line.
- Writes to the Brewfile and to symlink targets are atomic where the filesystem
  allows: write to a temp file then rename; create the new symlink under a temp
  name then rename over the old one.
- `--dry-run` applies to install, link, export, update and commit and prints
  the full plan with no side effects, including no tap trust, no brew upgrade
  and no git commands.

## 13. Testing

Unit tests, standard `go test`, using `t.TempDir()` as `$HOME` and as the repo:

- `manifest`: valid file loads; unknown key, missing `home/` source, duplicate
  id and non-`~/` target each produce a specific error.
- `link`: one test per row of the table in section 8, plus dry-run makes no
  change, plus backup path layout.
- `brew`: Brewfile parse round-trips lines with options; diff yields the right
  added and removed sets; filtered write preserves dump order; tap trust list
  is computed from the Brewfile.
- `steps`: check passes so run is skipped; check fails, run succeeds, recheck
  passes; recheck still failing marks failed; order is preserved; update runs
  only for steps that define it and ignores check.
- `update`: brew update and upgrade are invoked in order before any step
  update; a failing brew upgrade does not stop step updates.
- `services` and `manual`: status derived from scripted exit codes.
- `status`: aggregation and the `--json` shape.
- `tui`: model update logic for key handling and picker toggling, driven
  directly without a terminal.

Acceptance on this machine after migration:

1. `dot status` reports every link ok, all three services loaded, the Brewfile
   with nothing missing, and the manual items detectable here as done.
2. `dot install --dry-run` plans zero actions.
3. A fresh shell, tmux, nvim, Ghostty, yabai, skhd and sketchybar behave as
   before the migration.

## 14. One-time migration of this machine

Performed as the last implementation task, with a timestamped backup of every
touched live path.

1. Move live configs into `home/` (live versions win over repo copies):
   `~/.zshrc`, `~/.zprofile`, `~/.gitconfig`, `~/.config/{git,tmux,nvim,skhd,yabai,sketchybar}`,
   the Ghostty config, the lazygit `config.yml`. Track `lazy-lock.json` and
   `lazyvim.json` for reproducible plugin versions. Delete the vendored
   `sketchybar-app-font/` directory and gitignore `helper/helper`. This comes
   first because the manifest only validates once every link source exists,
   and `dot export` cannot run before the manifest loads.
2. Split `.zshrc`: move the credential exports to `~/.zshrc.local`, append the
   source lines from section 10 to the tracked file. Commit `home/` and the
   manifest only after confirming no credential is staged.
3. Trust the currently tapped taps, run the first `dot export`, add
   `cask "font-jetbrains-mono-nerd-font"` and `cask "font-sketchybar-app-font"`,
   commit the Brewfile.
4. Remove the duplicate `~/.sketchybarrc` (sketchybar reads
   `~/.config/sketchybar/sketchybarrc` first) and the redundant
   `~/.tmux.conf` symlink, since tmux 3.1+ reads `~/.config/tmux/tmux.conf`
   when `~/.tmux.conf` is absent. Both go to the backup directory.
5. `dot link`, then the acceptance checks in section 13.
6. Delete `linux/`, `macos/`, `shared/`, `themes/`. Write the README. Commit.

Known inert config kept as-is: `tmux.conf` declares `@plugin 'sainnhe/tmux-fzf'`
but no plugin manager is installed, so the line has no effect. Left for the
owner to decide; not part of this work.

## 15. Out of scope

- Linux support.
- Theme switching (`dot theme`).
- Pinning Homebrew package versions. Only nvim plugins are pinned, via `lazy-lock.json`.
- `brew bundle cleanup` or any uninstall.
- Self-update of the repo and the `dot` binary (`git pull` plus rebuild), and
  prebuilt release binaries.
- `brew upgrade --greedy` for self-updating casks; plain `brew upgrade` only.
- Managing `~/.zshrc.local` or any secret store.
