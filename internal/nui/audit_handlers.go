package nui

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nats-nui/nui/internal/connection"
	"github.com/nats-nui/nui/pkg/audit"
)

// AuditMiddleware logs API requests to the audit log
func AuditMiddleware(auditRepo audit.AuditRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip non-API routes and health checks
		path := c.Path()
		if !strings.HasPrefix(path, "/api/") || path == "/api/auth/me" {
			return c.Next()
		}

		// Skip GET requests for read operations (too noisy)
		method := c.Method()
		if method == "GET" {
			return c.Next()
		}

		// Call next handler first
		err := c.Next()

		// Get user info from locals (set by auth middleware)
		userID, _ := c.Locals("user_id").(string)
		userEmail, _ := c.Locals("user_email").(string)
		userName, _ := c.Locals("user_name").(string)

		// Determine action and resource from path and method
		action, resourceType, resourceID := parseRequest(method, path, c)

		// Extract topic for subscribe actions
		topic := ""
		if action == audit.ActionSubscribe {
			topic = extractTopicsFromBody(c)
		}

		// Log the action
		logEntry := audit.AuditLog{
			Timestamp:    time.Now(),
			UserID:       userID,
			UserEmail:    userEmail,
			UserName:     userName,
			Action:       action,
			Method:       method,
			Path:         path,
			StatusCode:   c.Response().StatusCode(),
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Topic:        topic,
		}

		// Fire and forget - don't block the response
		go func() {
			_ = auditRepo.Log(logEntry)
		}()

		return err
	}
}

// extractTopicsFromBody extracts subscription topics from the request body
func extractTopicsFromBody(c *fiber.Ctx) string {
	body := c.Body()
	if len(body) == 0 {
		return ""
	}

	var subs []connection.Subscription
	if err := json.Unmarshal(body, &subs); err != nil {
		return ""
	}

	// Collect all active subjects
	subjects := make([]string, 0, len(subs))
	for _, sub := range subs {
		if sub.Subject != "" && !sub.Disabled {
			subjects = append(subjects, sub.Subject)
		}
	}

	return strings.Join(subjects, ", ")
}

func parseRequest(method, path string, c *fiber.Ctx) (action, resourceType, resourceID string) {
	// Parse the path segments
	segments := strings.Split(strings.Trim(path, "/"), "/")

	// Default action based on method
	switch method {
	case "POST":
		action = audit.ActionCreate
	case "PUT", "PATCH":
		action = audit.ActionUpdate
	case "DELETE":
		action = audit.ActionDelete
	default:
		action = audit.ActionRead
	}

	// Parse resource type and ID from path
	// /api/connection/:id
	// /api/connection/:id/stream/:name
	// /api/connection/:id/kv/:bucket/key/:key
	if len(segments) >= 2 && segments[0] == "api" {
		switch segments[1] {
		case "connection":
			resourceType = audit.ResourceConnection
			if len(segments) >= 3 {
				resourceID = segments[2]
			}
			// Check for nested resources
			if len(segments) >= 4 {
				switch segments[3] {
				case "stream":
					resourceType = audit.ResourceStream
					if len(segments) >= 5 {
						resourceID = segments[4]
					}
					if len(segments) >= 6 && segments[5] == "consumer" {
						resourceType = audit.ResourceConsumer
						if len(segments) >= 7 {
							resourceID = segments[6]
						}
					}
					if len(segments) >= 6 && segments[5] == "messages" {
						resourceType = audit.ResourceMessage
					}
				case "kv":
					resourceType = audit.ResourceBucket
					if len(segments) >= 5 {
						resourceID = segments[4]
					}
					if len(segments) >= 6 && segments[5] == "key" {
						resourceType = audit.ResourceKey
						if len(segments) >= 7 {
							resourceID = segments[6]
						}
					}
				case "messages":
					if segments[4] == "publish" {
						action = audit.ActionPublish
						resourceType = audit.ResourceMessage
					} else if segments[4] == "subscription" {
						action = audit.ActionSubscribe
						resourceType = audit.ResourceMessage
					}
				case "request":
					action = audit.ActionRequest
					resourceType = audit.ResourceMessage
				}
			}
			// Check for import
			if len(segments) >= 4 && segments[3] == "import" {
				action = audit.ActionImport
			}
			// Check for purge
			if len(segments) >= 5 && segments[len(segments)-1] == "purge" {
				action = audit.ActionPurge
			}
		case "proto":
			resourceType = audit.ResourceProto
			if len(segments) >= 3 {
				resourceID = segments[2]
			}
		}
	}

	return action, resourceType, resourceID
}

// HandleListAuditLogs returns audit logs with optional filtering
func (a *App) HandleListAuditLogs(c *fiber.Ctx) error {
	filter := audit.ListFilter{}

	// Parse query parameters
	if userID := c.Query("user_id"); userID != "" {
		filter.UserID = userID
	}
	if action := c.Query("action"); action != "" {
		filter.Action = action
	}
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = &t
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filter.SortBy = sortBy
	}
	// Default to descending (newest first), only use ascending if explicitly set to false
	filter.SortDesc = c.Query("sort_desc") != "false"

	logs, err := a.nui.AuditRepo.List(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve audit logs: " + err.Error(),
		})
	}

	return c.JSON(logs)
}
