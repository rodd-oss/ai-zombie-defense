package types

import (
	"database/sql"
	"database/sql/driver"
	"time"
)

// Timestamp represents a SQLite timestamp stored as TEXT in ISO 8601 format.
type Timestamp struct {
	time.Time
}

// Scan implements sql.Scanner for Timestamp.
func (t *Timestamp) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case string:
		// Parse ISO 8601 format (e.g., "2026-01-22T15:51:23Z")
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	case []byte:
		return t.Scan(string(v))
	case time.Time:
		t.Time = v
		return nil
	default:
		return sql.ErrNoRows
	}
}

// Value implements driver.Valuer for Timestamp.
func (t Timestamp) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	// Format as ISO 8601 with UTC timezone
	return t.Time.UTC().Format(time.RFC3339), nil
}

// MarshalJSON implements json.Marshaler for Timestamp.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return t.Time.UTC().MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler for Timestamp.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	return t.Time.UnmarshalJSON(data)
}

// NullTimestamp represents a nullable SQLite timestamp.
type NullTimestamp struct {
	Timestamp
	Valid bool
}

// Scan implements sql.Scanner for NullTimestamp.
func (nt *NullTimestamp) Scan(value interface{}) error {
	if value == nil {
		nt.Timestamp = Timestamp{}
		nt.Valid = false
		return nil
	}
	nt.Valid = true
	return nt.Timestamp.Scan(value)
}

// Value implements driver.Valuer for NullTimestamp.
func (nt NullTimestamp) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Timestamp.Value()
}

// MarshalJSON implements json.Marshaler for NullTimestamp.
func (nt NullTimestamp) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return nt.Timestamp.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler for NullTimestamp.
func (nt *NullTimestamp) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		nt.Timestamp = Timestamp{}
		return nil
	}
	nt.Valid = true
	return nt.Timestamp.UnmarshalJSON(data)
}
