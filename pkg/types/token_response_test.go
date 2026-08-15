package types_test

import (
	"testing"

	"github.com/patbaumgartner/watchtower/pkg/types"
)

func TestTokenResponseBearerToken(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   types.TokenResponse
		want string
	}{
		{"token preferred", types.TokenResponse{Token: "a", AccessToken: "b"}, "a"},
		{"access_token fallback", types.TokenResponse{AccessToken: "b"}, "b"},
		{"neither", types.TokenResponse{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.BearerToken(); got != tc.want {
				t.Errorf("BearerToken() = %q, want %q", got, tc.want)
			}
		})
	}
}
