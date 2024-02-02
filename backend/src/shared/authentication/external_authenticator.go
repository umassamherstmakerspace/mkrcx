package leash_authentication

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/mkrcx/mkrcx/src/shared/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type ExternalAuthenticator interface {
	// GetAuthURL returns the URL to redirect the user to for authentication
	GetAuthURL(state string) string
	// Authenticate authenticates a user and returns the user's email
	Callback(ctx context.Context, code string) (string, error)
}

type GoogleAuthenticator struct {
	googleOauth oauth2.Config
}

var _ ExternalAuthenticator = (*GoogleAuthenticator)(nil)

func (g *GoogleAuthenticator) GetAuthURL(state string) string {
	return g.googleOauth.AuthCodeURL(state)
}

func (g *GoogleAuthenticator) Callback(ctx context.Context, code string) (string, error) {
	userinfo := &struct {
		Email string `json:"email" validate:"required,email"`
	}{}

	{
		// Exchange the code for a token
		tok, err := g.googleOauth.Exchange(ctx, code)
		if err != nil {
			return "", err
		}

		// Get the userinfo
		client := g.googleOauth.Client(ctx, tok)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		// Decode the userinfo
		err = json.NewDecoder(resp.Body).Decode(userinfo)
		if err != nil {
			return "", err
		}

		// Validate the userinfo
		if models.ValidateStruct(userinfo) != nil {
			return "", errors.New("invalid email")
		}
	}

	return userinfo.Email, nil
}

func GetGoogleAuthenticator(clientID string, clientSecret string, RedirectURL string) ExternalAuthenticator {
	return &GoogleAuthenticator{
		googleOauth: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  RedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
			},
			Endpoint: google.Endpoint,
		},
	}
}
