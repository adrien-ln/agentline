## What and why

<!-- One topic per pull request. Link the related issue, if any: Closes #… -->

## Checklist

- [ ] `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass
- [ ] `tests/golden/` is unchanged, or the change is intended and explained above (`go test ./internal/render -maj`, diff read)
- [ ] Every new user-visible string goes through `i18n.T`, with its English entry in `internal/i18n/catalogue_*.go`
- [ ] No new dependency (or it was discussed in an issue first)
- [ ] README, CLAUDE.md or `--help` text updated if the behavior changed; demos regenerated if they no longer match
- [ ] Commits follow `type(scope): summary` (see CONTRIBUTING.md)
