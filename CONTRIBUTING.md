# Contributing

## Requirements

- Docker. Every Go command here runs inside a pinned `golang:1.27.2-bookworm`
  or `golangci/golangci-lint:v2.14.0` image (`scripts/go-docker.sh`,
  `scripts/golangci-lint-docker.sh`) rather than whatever Go your host
  package manager has — see the "why" in the comments at the top of each
  script.
- [lefthook](https://github.com/evilmartians/lefthook) and
  [bun](https://bun.sh) (for commitlint). `bun install` then
  `lefthook install` once, after cloning.
- [gitleaks](https://github.com/gitleaks/gitleaks) on your `PATH`. It scans for
  secrets in the `pre-commit` and `pre-push` hooks and in the `gitleaks`
  workflow in CI, with the rules extended from the upstream defaults in
  `.gitleaks.toml`.

## Building and testing

```sh
./scripts/go-docker.sh go build ./...
./scripts/go-docker.sh go test -race ./...
./scripts/golangci-lint-docker.sh run ./...
./scripts/go-docker.sh sh -c "go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./..."
```

`lefthook run pre-push` runs the same set (plus `go mod tidy -diff` and a
gitleaks scan of the whole history) except `govulncheck`, which would fail every push the day an advisory lands on `main`.
Run that one by hand; CI owns the gate.

`scripts/build-reports.sh` turns `coverage.out` and `junit.xml` into the
published report layout; `scripts/build-reports.test.sh` checks it and runs
in the `test` job. The `pages-deploy` job in `ci.yml` publishes the result
on a push to `main`, after every required job is green. It needs GitHub
Pages set to the "GitHub Actions" source in the repo's settings.

## Regenerating the client

`internal/genclient/client.gen.go` is generated from `openapi/openapi.yaml`
by [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) — never hand-edit
it. The spec itself is pinned to a commit of
[alrayyes/forge-dashboard](https://github.com/alrayyes/forge-dashboard)
recorded in `openapi/SPEC_COMMIT`; `.github/workflows/regenerate.yml` bumps
that pin whenever forge-dashboard's own CI reports its spec changed (a
`repository_dispatch` sent on merge to its main), or weekly as a fallback
if that dispatch is ever missed. To do it by hand:

```sh
./hack/fetch-spec.sh
./scripts/go-docker.sh sh -c \
  "go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 \
   -generate types,client -package genclient \
   -o internal/genclient/client.gen.go openapi/openapi.yaml"
```

Diff `openapi/openapi.yaml` (not the generated Go) to decide whether a
change needs a major, minor or patch bump — `rules/sdk-generation.md`'s
"Versioning tracks the contract, not the commits". `.github/workflows/
regenerate.yml` already does this automatically with `oasdiff`, picking
`feat`/`fix` and adding a `BREAKING CHANGE:` footer as needed; a manual
regeneration should classify the diff the same way before writing the
commit message by hand.

## Known-pending secrets

Two repo secrets aren't set yet, matching every other new repo in this
account at this stage — both jobs stay red until they're added, which is
expected, not a bug to work around:

- `RELEASE_TOKEN` — a real-identity PAT `.github/workflows/release.yml`
  and `regenerate.yml` need (a `GITHUB_TOKEN`-authored PR can't self-merge
  and its checks can come back empty — see either workflow's own comment).
- `CODECOV_TOKEN` — kept out of the required `test` job specifically so
  its absence can't block every merge (`ci.yml`'s own comment).

## Commits and releases

Commit messages follow [Conventional
Commits](https://www.conventionalcommits.org/), linted by commitlint on
`commit-msg`. Merging to `main` is what
[release-please](https://github.com/googleapis/release-please) reads to
keep an open release pull request with the next version and changelog;
merging that pull request tags the release. There's no separate build step
— a tagged commit is what `go get` resolves.

## Branching

Every change lands through a pull request; nothing is pushed straight to
`main`.

## Dependency advisories

Two jobs share the work, both through
[advisory-scan](https://github.com/alrayyes/advisory-scan). The `audit` job in
`ci.yml` runs `govulncheck` on a pull request and on its base, and fails only on
advisories the pull request adds, so an advisory published overnight doesn't
turn unrelated pull requests red. It runs when a Go file, manifest or lockfile
changes, and on every push to `main`. `advisory-scan.yml` audits `main`
daily, on every push that changes a manifest or lockfile, and on demand, and
keeps one open issue titled "Dependency advisories" listing what it finds. It
closes that issue when the audit is clean. There's no ignore list: an advisory
with no fix stays in that issue until one ships.
