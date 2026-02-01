package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// AuthMiddleware checks if the user is authenticated
func AuthMiddleware(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip auth for public paths
		path := c.Path()
		publicPaths := []string{
			"/auth/",
			"/api/auth/me",
			"/health",
			"/login",
			"/assets/",
			"/favicon.ico",
		}

		for _, p := range publicPaths {
			if strings.HasPrefix(path, p) || path == p {
				return c.Next()
			}
		}

		// Check session
		sess, err := store.Get(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get session",
			})
		}

		authenticated := sess.Get("authenticated")
		if authenticated == nil || !authenticated.(bool) {
			// For API requests, return 401
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/") {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Unauthorized",
				})
			}
			// For page requests, redirect to login
			return c.Redirect("/login", fiber.StatusTemporaryRedirect)
		}

		// Store user info in locals for handlers
		c.Locals("user_id", sess.Get("user_id"))
		c.Locals("user_email", sess.Get("user_email"))
		c.Locals("user_name", sess.Get("user_name"))

		return c.Next()
	}
}

// OptionalAuthMiddleware adds user info to context if authenticated, but doesn't block
func OptionalAuthMiddleware(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.Next()
		}

		authenticated := sess.Get("authenticated")
		if authenticated != nil && authenticated.(bool) {
			c.Locals("user_id", sess.Get("user_id"))
			c.Locals("user_email", sess.Get("user_email"))
			c.Locals("user_name", sess.Get("user_name"))
			c.Locals("authenticated", true)
		}

		return c.Next()
	}
}
