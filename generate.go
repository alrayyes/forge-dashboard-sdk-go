// Package forgedashboard is the official Go SDK for forge-dashboard.
package forgedashboard

// client.gen.go is generated from forge-dashboard's pinned spec copy --
// never hand-edit it. See CONTRIBUTING.md's "Regenerating the client"
// section for the generated/hand-written boundary.
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0 -generate types,client -package genclient -o internal/genclient/client.gen.go openapi/openapi.yaml
