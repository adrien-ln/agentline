# Security policy

## Supported versions

Only the latest release receives fixes. Please check that the problem still happens with it (`agentline version`) before reporting.

## Reporting a vulnerability

Please report vulnerabilities privately, not in a public issue: on the repository's **Security** tab, choose **Report a vulnerability** (GitHub private vulnerability reporting). Include the version, your OS and the steps to reproduce. You will get an answer as soon as possible; a fix is published as a new release, with credit if you wish.

## Attack surface

- **The statusline** (`agentline render`, run by Claude Code at every event) reads the session JSON on stdin, your settings in `~/.config/agentline` and the state of the current Git repository (through `git`, with a short cache in the temporary folder). It makes no network call and writes nothing but that cache (and, once, a copy of the settings of the old `sl` name from `~/.config/statusline`).
- **`agentline install`** copies the binary, edits `~/.claude/settings.json`, the user PATH (registry on Windows, shell file elsewhere) and, after you answer yes, installs a Nerd Font (from the official nerd-fonts release, or through Scoop or Homebrew) and sets it in Windows Terminal and VS Code. Every configuration file it changes gets a dated `.bak` copy first. `agentline uninstall` only removes what the installation added.
- **`agentline doctor`** only reads (it may ask the terminal for its background color).
