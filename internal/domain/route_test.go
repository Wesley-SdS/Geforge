package domain

import (
	"net/url"
	"testing"
	"time"
)

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func TestRoute_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		route   Route
		wantErr bool
	}{
		{
			name: "valid route",
			route: Route{
				Path:            "/api/users",
				Methods:         []string{"GET", "POST"},
				Targets:         []Target{{URL: mustParseURL("http://localhost:3001"), Weight: 1}},
				BalanceStrategy: "round-robin",
				Timeout:         10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid route with weighted strategy",
			route: Route{
				Path:            "/api/orders",
				Targets:         []Target{{URL: mustParseURL("http://localhost:3001"), Weight: 3}},
				BalanceStrategy: "weighted",
			},
			wantErr: false,
		},
		{
			name:    "empty path",
			route:   Route{Targets: []Target{{URL: mustParseURL("http://localhost:3001"), Weight: 1}}},
			wantErr: true,
		},
		{
			name: "path without leading slash",
			route: Route{
				Path:    "api/users",
				Targets: []Target{{URL: mustParseURL("http://localhost:3001"), Weight: 1}},
			},
			wantErr: true,
		},
		{
			name:    "no targets",
			route:   Route{Path: "/api/users"},
			wantErr: true,
		},
		{
			name: "nil target URL",
			route: Route{
				Path:    "/api/users",
				Targets: []Target{{URL: nil, Weight: 1}},
			},
			wantErr: true,
		},
		{
			name: "negative weight",
			route: Route{
				Path:    "/api/users",
				Targets: []Target{{URL: mustParseURL("http://localhost:3001"), Weight: -1}},
			},
			wantErr: true,
		},
		{
			name: "invalid method",
			route: Route{
				Path:    "/api/users",
				Methods: []string{"INVALID"},
				Targets: []Target{{URL: mustParseURL("http://localhost:3001"), Weight: 1}},
			},
			wantErr: true,
		},
		{
			name: "invalid balance strategy",
			route: Route{
				Path:            "/api/users",
				Targets:         []Target{{URL: mustParseURL("http://localhost:3001"), Weight: 1}},
				BalanceStrategy: "random",
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			route: Route{
				Path:    "/api/users",
				Targets: []Target{{URL: mustParseURL("http://localhost:3001"), Weight: 1}},
				Timeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.route.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
