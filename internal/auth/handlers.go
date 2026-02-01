package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/nats-nui/nui/pkg/audit"
	"golang.org/x/oauth2"
)

type AuthHandlers struct {
	oidcConfig *OIDCConfig
	store      *session.Store
	auditRepo  audit.AuditRepo
}

func NewAuthHandlers(oidcConfig *OIDCConfig, store *session.Store, auditRepo audit.AuditRepo) *AuthHandlers {
	return &AuthHandlers{
		oidcConfig: oidcConfig,
		store:      store,
		auditRepo:  auditRepo,
	}
}

func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HandleLogin initiates the OIDC authorization code flow
func (h *AuthHandlers) HandleLogin(c *fiber.Ctx) error {
	// Ensure we're on localhost (not 127.0.0.1) to match the OAuth redirect URI
	host := c.Hostname()
	if host == "127.0.0.1" {
		return c.Redirect("http://localhost:"+c.Port()+"/auth/login", fiber.StatusFound)
	}

	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}

	state, err := generateState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate state",
		})
	}

	sess.Set("oauth_state", state)
	if err := sess.Save(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save session",
		})
	}

	authURL := h.oidcConfig.OAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return c.Redirect(authURL, fiber.StatusFound)
}

// HandleCallback processes the OIDC callback from JumpCloud
func (h *AuthHandlers) HandleCallback(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}

	// Get and URL-decode the state from query
	receivedState := c.Query("state")
	decodedState, err := url.QueryUnescape(receivedState)
	if err != nil {
		decodedState = receivedState
	}

	// Verify state
	savedState := sess.Get("oauth_state")
	if savedState == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No saved state in session - please try logging in again",
		})
	}
	if savedState.(string) != decodedState {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "State mismatch - please try logging in again",
		})
	}

	// Check for error from provider
	if errMsg := c.Query("error"); errMsg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             errMsg,
			"error_description": c.Query("error_description"),
		})
	}

	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing authorization code",
		})
	}

	// Exchange code for tokens
	ctx := c.Context()
	oauth2Token, err := h.oidcConfig.OAuth2Config.Exchange(ctx, code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to exchange token: " + err.Error(),
		})
	}

	// Extract and verify ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No id_token in response",
		})
	}

	idToken, err := h.oidcConfig.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify ID token: " + err.Error(),
		})
	}

	// Extract claims
	var claims UserClaims
	if err := idToken.Claims(&claims); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to parse claims: " + err.Error(),
		})
	}

	// Store user info and tokens in session
	sess.Set("authenticated", true)
	sess.Set("user_id", claims.Sub)
	sess.Set("user_email", claims.Email)
	sess.Set("user_name", claims.Name)
	sess.Set("access_token", oauth2Token.AccessToken)
	sess.Set("refresh_token", oauth2Token.RefreshToken)
	sess.Set("token_expiry", oauth2Token.Expiry.Unix())

	// Clear the oauth state
	sess.Delete("oauth_state")

	if err := sess.Save(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save session",
		})
	}

	// Log the login event
	if h.auditRepo != nil {
		go func() {
			_ = h.auditRepo.Log(audit.AuditLog{
				Timestamp:    time.Now(),
				UserID:       claims.Sub,
				UserEmail:    claims.Email,
				UserName:     claims.Name,
				Action:       audit.ActionLogin,
				Method:       "GET",
				Path:         "/auth/callback/jumpcloud",
				StatusCode:   fiber.StatusFound,
				ResourceType: audit.ResourceAuth,
			})
		}()
	}

	// Redirect to the app
	return c.Redirect("/", fiber.StatusFound)
}

// HandleLogout clears the session and logs out the user
func (h *AuthHandlers) HandleLogout(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}

	// Get user info before destroying session for audit log
	userID, _ := sess.Get("user_id").(string)
	userEmail, _ := sess.Get("user_email").(string)
	userName, _ := sess.Get("user_name").(string)

	if err := sess.Destroy(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to destroy session",
		})
	}

	// Log the logout event
	if h.auditRepo != nil {
		go func() {
			_ = h.auditRepo.Log(audit.AuditLog{
				Timestamp:    time.Now(),
				UserID:       userID,
				UserEmail:    userEmail,
				UserName:     userName,
				Action:       audit.ActionLogout,
				Method:       "GET",
				Path:         "/auth/logout",
				StatusCode:   fiber.StatusFound,
				ResourceType: audit.ResourceAuth,
			})
		}()
	}

	return c.Redirect("/login", fiber.StatusFound)
}

// HandleMe returns the current user's information
func (h *AuthHandlers) HandleMe(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}

	authenticated := sess.Get("authenticated")
	if authenticated == nil || !authenticated.(bool) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"authenticated": false,
		})
	}

	return c.JSON(fiber.Map{
		"authenticated": true,
		"user": fiber.Map{
			"id":    sess.Get("user_id"),
			"email": sess.Get("user_email"),
			"name":  sess.Get("user_name"),
		},
	})
}

// RefreshTokenIfNeeded checks if the token is about to expire and refreshes it
func (h *AuthHandlers) RefreshTokenIfNeeded(sess *session.Session) error {
	expiry := sess.Get("token_expiry")
	if expiry == nil {
		return nil
	}

	expiryTime := time.Unix(expiry.(int64), 0)
	if time.Until(expiryTime) > 5*time.Minute {
		return nil
	}

	refreshToken := sess.Get("refresh_token")
	if refreshToken == nil {
		return nil
	}

	token := &oauth2.Token{
		RefreshToken: refreshToken.(string),
	}

	tokenSource := h.oidcConfig.OAuth2Config.TokenSource(nil, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return err
	}

	sess.Set("access_token", newToken.AccessToken)
	if newToken.RefreshToken != "" {
		sess.Set("refresh_token", newToken.RefreshToken)
	}
	sess.Set("token_expiry", newToken.Expiry.Unix())

	return nil
}
