# History: from bash to Go

agentline started as a bash script (`statusline.sh`) with a `sl-theme` picker, also in bash. In October 2026 it was rewritten in Go, then renamed from `sl` (project `statusline`) to `agentline`. This page keeps the decisions behind the rewrite and how it was checked.

## Why rewrite

The bash version worked: 30 to 65 ms per render on Windows, well under the 300 ms Claude Code waits between updates, and 184 tests with byte-for-byte reference outputs. It was stuck on three points:

- **macOS** ships bash 3.2: no `mapfile`, `declare -A`, `${x,,}` or `printf '%(…)T'`. The script needed bash 5 from Homebrew.
- **An animated picker** (a fake Claude Code session typed with a human rhythm, a mascot) is possible in bash but fragile: animation loop, `read -t` for the keyboard, redrawing without spawning processes.
- **Two languages** (bash plus a PowerShell profile function) to keep in sync.

## Options considered

| Option | Verdict |
|---|---|
| Keep bash | Rendering fine, but macOS and the animation stay hard |
| Hybrid: bash statusline, compiled picker | Two languages, and macOS still blocked on the statusline |
| **All in Go** | Chosen: one binary, native on the three systems, Bubble Tea for the TUI |
| All in Rust | Comparable, but slower to write and cross-compile for no gain here |
| Node, Python, PowerShell | Excluded: 200 ms or more per start, on every Claude Code event |

Go won over Rust for cross-compilation in one command and for Bubble Tea, which covers the picker exactly. On Windows, any process costs about 20 ms to start (`git.exe --version`: 22.5 ms), so the binary size and language speed matter less than startup.

## Gate: a prototype first

The rewrite had a stop condition: if the Go prototype did not render at least as fast as bash, the project would fall back to the hybrid option.

| Full render | Windows | Linux |
|---|---|---|
| bash `statusline.sh` | 37.1 ms | 8.2 ms |
| Go | **18.6 ms** | **2.7 ms** |

On Windows the Go version is close to the minimal cost of starting a program.

## Byte-for-byte compatibility

The Go renderer reproduces the bash functions one by one, including the JSON reading by regular expressions, so a malformed JSON is tolerated the same way (display what can be read rather than nothing). It shares the same 5 s Git cache.

- The 30 bash reference outputs in `tests/golden/` pass unchanged.
- Before the bash version was removed, every theme was rendered on 3 cases and compared with the real `statusline.sh`, then frozen as references (`tests/golden/themes/`). Bash tests without a Go equivalent were rewritten: theme format and tags, invalid names, fallback palette, remotes, Windows paths, worktrees, settings priority.
- Later additions (256, 16 and no-color levels, light background, English) have their own references, produced by Go and reviewed, in subfolders of `tests/golden/`.
- `go test ./internal/render -maj` rewrites the references after an intended change.

## Design decisions

- **One binary, embedded themes** (`go:embed`); user themes are read only if they exist. The hot path (`agentline render`) reads the JSON, the chosen theme and the Git cache, nothing else.
- **Previews in the same process**: the picker renders every theme with a fake Git state, in under a millisecond (bash: about 75 ms for an unseen theme). No temporary repository, no subprocess.
- **Deterministic animation**: the scenes are timelines replayed by a timer, with a fixed-seed generator for the irregular typing (45 to 135 ms per character, visible typos corrected, 1.5 to 3.4 s of thinking). The state at time t is computed by replaying the timeline, so every frame is testable. About 45 µs per frame at 20 frames per second, only while the picker is open.
- **Clawd**, the mascot, uses the real Claude Code drawing (3 × 9 blocks) in the theme colors. Since then it has been replaced by Pip, an original character: Clawd's drawing belongs to Anthropic.
- **Non-intrusive installation**: `agentline install` writes the `statusLine` key in `~/.claude/settings.json`, validates the JSON with `encoding/json` and refuses to leave it invalid, makes a dated `.bak` copy before every change and changes nothing when everything is up to date. Everything that touches the system goes through injectable options, so the tests simulate Windows, macOS and Linux on any machine.
- **No network at runtime**, no CGO, dependencies pinned in `go.sum`, `govulncheck` in the CI.
- **PowerShell**: `sl` is a built-in alias of `Set-Location`, one more reason for the rename. `agentline` alone now opens the picker; the `sl-theme` command and the profile function are gone.

## The rename

`sl` became `agentline` (module `github.com/adrien-ln/agentline`). Settings moved to `~/.config/agentline` (copied once from `~/.config/statusline`), `SL_*` variables became `AGENTLINE_*` (the old names are still read, deprecated), and `agentline install` sets the old binary and commands aside as `.bak` copies.
