package types

// TokenResponse is returned by the registry on successful authentication.
// The Docker registry token spec accepts "access_token" as an OAuth2-compatible
// alias for "token"; some registries only send the former.
type TokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
}

// BearerToken returns the token to present to the registry, or an empty string
// if the response carried none.
func (r TokenResponse) BearerToken() string {
	if r.Token != "" {
		return r.Token
	}
	return r.AccessToken
}
