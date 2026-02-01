package audit

import (
	"time"

	docstore "github.com/nats-nui/nui/pkg/storage"
	"github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
)

const AUDIT_COLLECTION = "audit_logs"

type AuditRepo interface {
	Log(log AuditLog) error
	List(filter ListFilter) ([]AuditLog, error)
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
	doc := r.db.DocFromType(log)
	_, err := r.db.InsertOne(AUDIT_COLLECTION, doc)
	return err
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

	// Sort by specified field (default: timestamp descending)
	sortField := filter.SortBy
	if sortField == "" {
		sortField = "timestamp"
	}
	sortDirection := -1 // descending
	if !filter.SortDesc {
		sortDirection = 1 // ascending
	}
	q = q.Sort(query.SortOption{Field: sortField, Direction: sortDirection})

	// Apply limit
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	q = q.Limit(limit)

	// Apply offset
	if filter.Offset > 0 {
		q = q.Skip(filter.Offset)
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

	return logs, nil
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
