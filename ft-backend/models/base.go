package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DefaultTenantID is the UUID for the default tenant.
// All records belong to this tenant until multi-tenant is fully enabled.
const DefaultTenantID = "00000000-0000-0000-0000-000000000001"

// BaseModel contains common fields shared by all entities.
// Uses UUID primary key and includes a tenant_id for future multi-tenant support.
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SoftDeleteModel extends BaseModel with soft-delete support.
type SoftDeleteModel struct {
	BaseModel
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// ---------------------------------------------------------------------------
// JSONB – a flexible PostgreSQL JSONB column type backed by json.RawMessage
// ---------------------------------------------------------------------------

// JSONB represents a PostgreSQL JSONB column.
// It can hold any valid JSON value: object, array, string, number, bool, null.
type JSONB json.RawMessage

// GormDataType tells GORM to use the "jsonb" column type.
func (j JSONB) GormDataType() string {
	return "jsonb"
}

// Value implements driver.Valuer for database writes.
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// Scan implements sql.Scanner for database reads.
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = JSONB("null")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append(JSONB{}, v...)
	case string:
		*j = JSONB(v)
	default:
		return fmt.Errorf("JSONB.Scan: unsupported type %T", value)
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONB) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("JSONB.UnmarshalJSON: receiver is nil")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// String returns the raw JSON string.
func (j JSONB) String() string {
	return string(j)
}

// NewJSONBFromMap creates a JSONB value from a map.
func NewJSONBFromMap(m map[string]interface{}) JSONB {
	b, _ := json.Marshal(m)
	return JSONB(b)
}

// NewJSONBFromSlice creates a JSONB value from a slice.
func NewJSONBFromSlice(s interface{}) JSONB {
	b, _ := json.Marshal(s)
	return JSONB(b)
}

// MustParseUUID parses a UUID string and panics on failure.
func MustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("models.MustParseUUID: invalid UUID %q: %v", s, err))
	}
	return id
}
