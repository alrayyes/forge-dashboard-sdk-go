package forgedashboard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	forgedashboard "github.com/alrayyes/forge-dashboard-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func authHeaderEchoServer(t *testing.T, gotHeader *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"version":"dev"}`))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestTokenSource(t *testing.T) {
	// Not parallel: t.Setenv mutates process-global state the "env
	// fallback" case depends on, which a concurrent subtest would race.
	cases := map[string]struct {
		opts []forgedashboard.Option
		env  string
		want string
	}{
		"option only":          {opts: []forgedashboard.Option{forgedashboard.WithToken("from-option")}, want: "Bearer from-option"},
		"env fallback":         {env: "from-env", want: "Bearer from-env"},
		"option overrides env": {opts: []forgedashboard.Option{forgedashboard.WithToken("from-option")}, env: "from-env", want: "Bearer from-option"},
		"neither set":          {want: ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv(forgedashboard.TokenEnvVar, tc.env)
			}

			var gotHeader string
			srv := authHeaderEchoServer(t, &gotHeader)

			client, err := forgedashboard.New(srv.URL, tc.opts...)
			require.NoError(t, err)

			_, err = client.GetVersionWithResponse(context.Background())
			require.NoError(t, err)

			assert.Equal(t, tc.want, gotHeader)
		})
	}
}

func TestDecodeError(t *testing.T) {
	t.Parallel()

	t.Run("below 400 is nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, forgedashboard.DecodeError(http.StatusOK, nil))
	})

	t.Run("404 decodes the error body", func(t *testing.T) {
		t.Parallel()
		err := forgedashboard.DecodeError(http.StatusNotFound, []byte(`{"error":"no user is registered under that username"}`))
		require.NotNil(t, err)
		assert.Equal(t, http.StatusNotFound, err.StatusCode)
		assert.Equal(t, "no user is registered under that username", err.Message)
		assert.Contains(t, err.Error(), "404")
	})
}
