package audit

import "time"

type AuditLog struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	TimestampMs  int64                  `json:"timestamp_ms"` // Unix milliseconds for reliable sorting
	UserID       string                 `json:"user_id"`
	UserEmail    string                 `json:"user_email"`
	UserName     string                 `json:"user_name"`
	Action       string                 `json:"action"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	StatusCode   int                    `json:"status_code"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Topic        string                 `json:"topic,omitempty"` // NATS subject for subscribe actions
	Details      map[string]interface{} `json:"details,omitempty"`
}

// Common action types
const (
	ActionLogin          = "login"
	ActionLogout         = "logout"
	ActionCreate         = "create"
	ActionRead           = "read"
	ActionUpdate         = "update"
	ActionDelete         = "delete"
	ActionPurge          = "purge"
	ActionPublish        = "publish"
	ActionSubscribe      = "subscribe"
	ActionRequest        = "request"
	ActionImport         = "import"
)

// Common resource types
const (
	ResourceConnection = "connection"
	ResourceStream     = "stream"
	ResourceConsumer   = "consumer"
	ResourceBucket     = "bucket"
	ResourceKey        = "key"
	ResourceMessage    = "message"
	ResourceProto      = "proto"
	ResourceAuth       = "auth"
)
