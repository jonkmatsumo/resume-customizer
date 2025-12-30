package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// User represents a user profile
type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never serialize to JSON
	PasswordSet  bool      `json:"password_set" db:"password_set"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Job represents an employment history entry
type Job struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Company        string    `json:"company"`
	RoleTitle      string    `json:"role_title"`
	Location       string    `json:"location,omitempty"`
	EmploymentType string    `json:"employment_type"` // full-time, part-time, etc.
	StartDate      *Date     `json:"start_date,omitempty"`
	EndDate        *Date     `json:"end_date,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Experience represents a bullet point within a job
type Experience struct {
	ID               uuid.UUID   `json:"id"`
	JobID            uuid.UUID   `json:"job_id"`
	BulletText       string      `json:"bullet_text"`
	Skills           StringArray `json:"skills"` // JSONB array
	EvidenceStrength string      `json:"evidence_strength"`
	RiskFlags        StringArray `json:"risk_flags"` // JSONB array
	CreatedAt        time.Time   `json:"created_at"`
}

// Education represents an education entry
type Education struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	School     string    `json:"school"`
	DegreeType string    `json:"degree_type,omitempty"`
	Field      string    `json:"field,omitempty"`
	GPA        string    `json:"gpa,omitempty"`
	Location   string    `json:"location,omitempty"`
	StartDate  *Date     `json:"start_date,omitempty"`
	EndDate    *Date     `json:"end_date,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Date is a custom type for handling SQL DATE (YYYY-MM-DD)
type Date struct {
	time.Time
}

// Scan implements the Scanner interface
func (d *Date) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	t, ok := value.(time.Time)
	if !ok {
		return errors.New("failed to scan Date")
	}
	d.Time = t
	return nil
}

// Value implements the Valuer interface
func (d *Date) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	return d.Time, nil
}

// MarshalJSON implements json.Marshaler
func (d *Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Format("2006-01-02"))
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Date) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == "null" || str == `""` {
		return nil
	}
	// Trim quotes
	if len(str) > 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	var err error
	d.Time, err = time.Parse("2006-01-02", str)
	return err
}

// StringArray handles JSONB string arrays
type StringArray []string

// Scan implements the Scanner interface for StringArray
func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = []string{}
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed")
	}
	return json.Unmarshal(source, a)
}

// Value implements the Valuer interface for StringArray
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(a)
}
