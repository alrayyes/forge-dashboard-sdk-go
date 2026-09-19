# Contributing

## Requirements

- Docker. Every Go command here runs inside a pinned `golang:1.27.1-bookworm`
  or `golangci/golangci-lint:v2.13.2` image (`scripts/go-docker.sh`,
  `scripts/golangci-lint-docker.sh`) rather than whatever Go your host
  package manager has — see the "why" in the comments at the top of each
  script.
- [lefthook](https://github.com/evilmartians/lefthook) and
  [bun](https://bun.sh) (for commitlint). `bun install` then
  `lefthook install` once, after cloning.

## Building and testing

```sh
./scripts/go-docker.sh go build ./...
./scripts/go-docker.sh go test -race ./...
./scripts/golangci-lint-docker.sh run ./...
./scripts/go-docker.sh sh -c "go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./..."
```

`lefthook run pre-push` runs the same set (plus `go mod tidy -diff`), so
that's the one command to run before opening a pull request.

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
