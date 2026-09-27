//go:build contract

// Package contract runs the client against a Prism mock server generated
// from forge-dashboard's own pinned spec (see .github/workflows/ci.yml's
// "contract" job) -- never a hand-rolled stub, per
// rules/sdk-generation.md's "testing against the spec, not a hand-written
// stub". This proves the client's requests/responses conform to the
// spec; it says nothing about whether the real server still matches that
// spec.
package contract

import (
	"context"
	"net/http"
	"os"
	"testing"

	forgedashboard "github.com/alrayyes/forge-dashboard-sdk-go"
)

func mustClient(t *testing.T) *forgedashboard.Client {
	t.Helper()
	baseURL := os.Getenv("FORGE_DASHBOARD_BASE_URL")
	if baseURL == "" {
		t.Fatal("FORGE_DASHBOARD_BASE_URL must point at a running Prism mock (see ci.yml's contract job)")
	}
	client, err := forgedashboard.New(baseURL, forgedashboard.WithToken("prism-does-not-check-this"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return client
}

func TestContract_Health(t *testing.T) {
	client := mustClient(t)
	resp, err := client.HealthWithResponse(context.Background())
	if err != nil {
		t.Fatalf("HealthWithResponse: %v", err)
	}
	if resp.HTTPResponse.StatusCode != http.StatusOK {
		t.Fatalf("HealthWithResponse: status %d, body %s", resp.HTTPResponse.StatusCode, resp.Body)
	}
}

func TestContract_GetSettings(t *testing.T) {
	client := mustClient(t)
	resp, err := client.GetSettingsWithResponse(context.Background())
	if err != nil {
		t.Fatalf("GetSettingsWithResponse: %v", err)
	}
	if resp.HTTPResponse.StatusCode != http.StatusOK {
		t.Fatalf("GetSettingsWithResponse: status %d, body %s", resp.HTTPResponse.StatusCode, resp.Body)
	}
}

func TestContract_ListCredentials(t *testing.T) {
	client := mustClient(t)
	resp, err := client.ListCredentialsWithResponse(context.Background())
	if err != nil {
		t.Fatalf("ListCredentialsWithResponse: %v", err)
	}
	if resp.HTTPResponse.StatusCode != http.StatusOK {
		t.Fatalf("ListCredentialsWithResponse: status %d, body %s", resp.HTTPResponse.StatusCode, resp.Body)
	}
}

func TestContract_ListUsers(t *testing.T) {
	client := mustClient(t)
	resp, err := client.ListUsersWithResponse(context.Background())
	if err != nil {
		t.Fatalf("ListUsersWithResponse: %v", err)
	}
	if resp.HTTPResponse.StatusCode != http.StatusOK {
		t.Fatalf("ListUsersWithResponse: status %d, body %s", resp.HTTPResponse.StatusCode, resp.Body)
	}
}
