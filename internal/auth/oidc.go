package auth

import (
	"context"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDCConfig struct {
	Provider     *oidc.Provider
	OAuth2Config *oauth2.Config
	Verifier     *oidc.IDTokenVerifier
}

func NewOIDCConfig(ctx context.Context) (*OIDCConfig, error) {
	issuer := os.Getenv("JUMPCLOUD_ISSUER")
	if issuer == "" {
		issuer = "https://oauth.id.jumpcloud.com/"
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	clientID := os.Getenv("JUMPCLOUD_CLIENT_ID")
	clientSecret := os.Getenv("JUMPCLOUD_CLIENT_SECRET")
	redirectURI := os.Getenv("JUMPCLOUD_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:31311/auth/callback/jumpcloud"
	}

	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	return &OIDCConfig{
		Provider:     provider,
		OAuth2Config: oauth2Config,
		Verifier:     verifier,
	}, nil
}

type UserClaims struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
}
