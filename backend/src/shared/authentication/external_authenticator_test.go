package leash_authentication

import (
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestGoogleAuthenticationAlwaysPromptsForAccount(t *testing.T) {
	authenticator := &GoogleAuthenticator{
		googleOauth: oauth2.Config{
			ClientID: "test-client",
			Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.example/auth"},
		},
	}

	parsed, err := url.Parse(authenticator.GetAuthURL("signed-state"))
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Query().Get("prompt"); got != "select_account" {
		t.Fatalf("prompt = %q, want select_account", got)
	}
	if got := parsed.Query().Get("state"); got != "signed-state" {
		t.Fatalf("state = %q, want signed-state", got)
	}
}
