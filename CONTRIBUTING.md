# Contributing to agentline

Thanks for your interest. Bug reports, themes and fixes are welcome; for a larger change, open an issue first so we can agree on the approach.

## Setup

You need Go 1.26 or later. No other tool is required to build and test.

```bash
git clone https://github.com/adrien-ln/agentline.git && cd agentline
gofmt -l . ; go vet ./... && go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...   # also run by the CI
```

`gofmt -l .` must print nothing. [CLAUDE.md](CLAUDE.md) maps the code and lists the project rules; [docs/HISTORY.md](docs/HISTORY.md) explains the main design decisions.

Never run `agentline install` or `agentline uninstall` to try a change: they edit your real PATH, shell files and `~/.claude/settings.json`. The installer is tested through simulations in `internal/installation`. For manual tests of `theme` or `style` commands, point `AGENTLINE_CONFIG` at a temporary folder so your own settings stay untouched:

```bash
AGENTLINE_CONFIG=$(mktemp -d) go run ./cmd/agentline theme set nord
```

## Reference outputs

The statusline is checked byte for byte against the files in `tests/golden/`. A change that alters one of them by accident is a regression.

If you change the rendering on purpose, regenerate them and read the diff before committing:

```bash
go test ./internal/render -maj
git diff tests/golden/
```

Explain in the pull request why the references changed.

## Text and languages

agentline speaks English and French. Every user-visible string goes through the catalog: write `i18n.T("texte français")` in the code and add its English translation in `internal/i18n/catalogue_*.go`. `TestTextesDuCode` fails on a missing entry, an unused one, or mismatched `%` verbs. Code and comments are written in French; the public interface (commands, options, style keys, environment variables) is in English.

## Dependencies

The project has two direct dependencies (Bubble Tea and go-runewidth, plus `golang.org/x/sys`). Please do not add a new one without discussing it in an issue first.

## Commits and pull requests

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/), in English:

```
type(scope): imperative summary in lowercase
```

- Types: `feat`, `fix`, `perf`, `refactor`, `test`, `docs`, `chore`, `ci`, `build`, `revert`.
- 72 characters at most, no final period; an optional body, after a blank line, explains why.
- Scopes: `render`, `selector`, `style`, `theme`, `install`, `cli`, `i18n`, `colors`, `docs`, `ci`, `deps`, or none for general changes.
- Keep commits small; separate refactoring from behavior changes.

The CI checks every commit of a pull request, and its title, with `scripts/verifier-commits.sh`. Run it before pushing, or use it as a `commit-msg` hook:

```bash
bash scripts/verifier-commits.sh origin/main..HEAD
printf '#!/bin/sh\nexec bash scripts/verifier-commits.sh --file "$1"\n' > .git/hooks/commit-msg && chmod +x .git/hooks/commit-msg
```

One pull request covers one topic. Fill in the checklist of the template.

## Releasing (maintainers)

1. On an up-to-date `main` with a green CI, preview the notes: `bash scripts/notes-de-version.sh HEAD`.
2. In `CHANGELOG.md`, move `[Unreleased]` to `[X.Y.Z] - YYYY-MM-DD`, commit `docs: prepare vX.Y.Z`, push.
3. Optionally build locally to check: `bash scripts/construire.sh X.Y.Z` (output in `dist/X.Y.Z/`).
4. Tag and push: `git tag -a vX.Y.Z -m "agentline vX.Y.Z" && git push origin vX.Y.Z`.
5. `.github/workflows/release.yml` tests, builds the archives, attests and publishes the release, then pushes `agentline.json` to [scoop-agentline](https://github.com/adrien-ln/scoop-agentline) and `agentline.rb` to [homebrew-agentline](https://github.com/adrien-ln/homebrew-agentline) with the `PACKAGING_TOKEN` secret (a fine-grained token with Contents: write on those two repositories). Without the secret, copy them by hand.

## Demos

The GIF and the image of the README are made with [vhs](https://github.com/charmbracelet/vhs). If your change affects what they show, regenerate them:

```bash
vhs docs/demo/demo.tape && vhs docs/demo/statusline.tape
```

## Security

Please do not report vulnerabilities in public issues: see [SECURITY.md](SECURITY.md).
