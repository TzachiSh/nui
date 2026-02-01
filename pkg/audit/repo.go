package audit

import (
	"sort"
	"strings"
	"time"

	docstore "github.com/nats-nui/nui/pkg/storage"
	"github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
)

const AUDIT_COLLECTION = "audit_logs"

type AuditRepo interface {
	Log(log AuditLog) error
	List(filter ListFilter) ([]AuditLog, error)
	LogSubscriptionExpiry(userID, subject, reason string) error
}

type ListFilter struct {
	UserID     string
	Action     string
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
	SortBy     string // "timestamp", "action", "user_name", "user_email"
	SortDesc   bool   // true for descending, false for ascending
}

type DocStoreAuditRepo struct {
	db *docstore.DB
}

func NewDocStoreAuditRepo(db *docstore.DB) (*DocStoreAuditRepo, error) {
	// Ensure collection exists
	ok, err := db.HasCollection(AUDIT_COLLECTION)
	if err != nil {
		return nil, err
	}
	if !ok {
		if err := db.CreateCollection(AUDIT_COLLECTION); err != nil {
			return nil, err
		}
	}
	return &DocStoreAuditRepo{db: db}, nil
}

func (r *DocStoreAuditRepo) Log(log AuditLog) error {
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}
	// Set numeric timestamp for reliable sorting
	log.TimestampMs = log.Timestamp.UnixMilli()
	doc := r.db.DocFromType(log)
	_, err := r.db.InsertOne(AUDIT_COLLECTION, doc)
	return err
}

// LogSubscriptionExpiry logs a subscription expiry event to the audit log
func (r *DocStoreAuditRepo) LogSubscriptionExpiry(userID, subject, reason string) error {
	log := AuditLog{
		Timestamp:    time.Now(),
		UserID:       userID,
		Action:       ActionUnsubscribe,
		Method:       "SYSTEM",
		Path:         "/ws/subscription/expired",
		StatusCode:   200,
		ResourceType: ResourceMessage,
		Topic:        subject,
		Details: map[string]interface{}{
			"reason": reason,
		},
	}
	return r.Log(log)
}

func (r *DocStoreAuditRepo) List(filter ListFilter) ([]AuditLog, error) {
	q := query.NewQuery(AUDIT_COLLECTION)

	// Apply filters
	var criteria query.Criteria

	if filter.UserID != "" {
		c := query.Field("user_id").Eq(filter.UserID)
		if criteria == nil {
			criteria = c
		} else {
			criteria = criteria.And(c)
		}
	}

	if filter.Action != "" {
		c := query.Field("action").Eq(filter.Action)
		if criteria == nil {
			criteria = c
		} else {
			criteria = criteria.And(c)
		}
	}

	if filter.StartTime != nil {
		c := query.Field("timestamp").GtEq(filter.StartTime.Format(time.RFC3339))
		if criteria == nil {
			criteria = c
		} else {
			criteria = criteria.And(c)
		}
	}

	if filter.EndTime != nil {
		c := query.Field("timestamp").LtEq(filter.EndTime.Format(time.RFC3339))
		if criteria == nil {
			criteria = c
		} else {
			criteria = criteria.And(c)
		}
	}

	if criteria != nil {
		q = q.Where(criteria)
	}

	docs, err := r.db.FindAll(q)
	if err != nil {
		return nil, err
	}

	logs := make([]AuditLog, 0, len(docs))
	for _, doc := range docs {
		log, err := unmarshalAuditDoc(doc)
		if err != nil {
			continue // Skip malformed records
		}
		logs = append(logs, *log)
	}

	// Sort in Go for reliable sorting
	sortField := filter.SortBy
	if sortField == "" {
		sortField = "timestamp"
	}
	sortLogs(logs, sortField, filter.SortDesc)

	// Apply offset and limit after sorting
	if filter.Offset > 0 && filter.Offset < len(logs) {
		logs = logs[filter.Offset:]
	} else if filter.Offset >= len(logs) {
		logs = []AuditLog{}
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit < len(logs) {
		logs = logs[:limit]
	}

	return logs, nil
}

func sortLogs(logs []AuditLog, field string, desc bool) {
	sort.Slice(logs, func(i, j int) bool {
		var less bool
		switch field {
		case "timestamp":
			less = logs[i].Timestamp.Before(logs[j].Timestamp)
		case "user_name":
			less = strings.ToLower(logs[i].UserName) < strings.ToLower(logs[j].UserName)
		case "user_email":
			less = strings.ToLower(logs[i].UserEmail) < strings.ToLower(logs[j].UserEmail)
		case "action":
			less = strings.ToLower(logs[i].Action) < strings.ToLower(logs[j].Action)
		default:
			less = logs[i].Timestamp.Before(logs[j].Timestamp)
		}
		if desc {
			return !less
		}
		return less
	})
}

func unmarshalAuditDoc(doc *document.Document) (*AuditLog, error) {
	log := &AuditLog{}
	err := doc.Unmarshal(log)
	if err != nil {
		return nil, err
	}
	log.ID = doc.ObjectId()
	return log, nil
}
