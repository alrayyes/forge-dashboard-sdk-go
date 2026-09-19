# forge-dashboard-sdk-go

[![CI](https://github.com/alrayyes/forge-dashboard-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/forge-dashboard-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/alrayyes/forge-dashboard-sdk-go.svg)](https://pkg.go.dev/github.com/alrayyes/forge-dashboard-sdk-go)
[![Codecov](https://codecov.io/gh/alrayyes/forge-dashboard-sdk-go/graph/badge.svg)](https://codecov.io/gh/alrayyes/forge-dashboard-sdk-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go client for [forge-dashboard](https://github.com/alrayyes/forge-dashboard)'s
REST API, generated from its OpenAPI spec with
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). It saves you
from hand-rolling HTTP requests, retries and pagination against the API
yourself.

## Requirements

- Go 1.27 or later.
- A running forge-dashboard instance.
- A personal API token for that instance (see "Authentication" below) —
  every endpoint except `/healthz` and `/api/version` needs one.

## Installation

```sh
go get github.com/alrayyes/forge-dashboard-sdk-go@latest
```

Pin an exact tagged version (`@v0.1.0`) once one exists, rather than tracking
`@latest` in anything but a quick trial.

## Authentication

forge-dashboard authenticates browsers with passkeys (WebAuthn) and a
session cookie, but its spec documents a real headless alternative: a
personal API token, generated from the dashboard's own Settings page
(`POST /api/tokens`, signed in as yourself first) and sent as
`Authorization: Bearer <token>` on every request after. Pass it to `New`
or set `FORGE_DASHBOARD_TOKEN`:

```go
client, err := forgedashboard.New("https://dashboard.example.com",
    forgedashboard.WithToken(os.Getenv("FORGE_DASHBOARD_TOKEN")),
)
```

A token is revocable from Settings at any point; there's nothing in this
SDK to refresh one automatically once it's gone.

## Usage

`GetVersion` needs no token and is a good first call to prove the client
reaches the server at all:

```go
package main

import (
	"context"
	"fmt"
	"log"

	forgedashboard "github.com/alrayyes/forge-dashboard-sdk-go"
)

func main() {
	client, err := forgedashboard.New("https://dashboard.example.com")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.GetVersionWithResponse(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("server version:", resp.JSON200.Version)
}
```

Fetching your own dashboard needs a token:

```go
client, err := forgedashboard.New(
	"https://dashboard.example.com",
	forgedashboard.WithToken(os.Getenv("FORGE_DASHBOARD_TOKEN")),
)
if err != nil {
	log.Fatal(err)
}

resp, err := client.GetDashboardWithResponse(context.Background(), nil)
if err != nil {
	log.Fatal(err) // transport failure -- already retried
}
if apiErr := forgedashboard.DecodeError(resp.HTTPResponse.StatusCode, resp.Body); apiErr != nil {
	log.Fatal(apiErr)
}
for _, pr := range resp.JSON200.PullRequests {
	fmt.Printf("%s#%d: %s (%s)\n", pr.Repo, pr.Number, pr.Title, pr.Ci)
}
```

Every other operation follows the generated `ClientWithResponses` pattern —
`client.<Operation>WithResponse(ctx, ...)` returns a typed response whose
`JSON200`/`JSON4xx` fields hold the decoded body. `DecodeError` turns a
failed response into a `*forgedashboard.APIError` uniformly, as shown above.

The client retries a `429` or `5xx` response with exponential backoff and
jitter (honoring a server-sent `Retry-After`), and never retries any other
`4xx`. Tune it with `WithRetry`, or swap the underlying `*http.Client`
entirely with `WithHTTPClient`.

## Regenerating the client

See [CONTRIBUTING.md](CONTRIBUTING.md) — the generated code is pinned to a
specific forge-dashboard commit and shouldn't drift from it silently.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for building, testing and the release
process.

## License

[MIT](LICENSE) — a permissive license for the client, independent of
forge-dashboard's own AGPL-3.0.
