package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Custom Types & Constants
// ---------------------------------------------------------------------------

// HabitStatus represents the lifecycle state of a habit.
type HabitStatus string

const (
	HabitStatusActive   HabitStatus = "ACTIVE"
	HabitStatusPaused   HabitStatus = "PAUSED"
	HabitStatusArchived HabitStatus = "ARCHIVED"
)

// CompletionType represents how a habit log entry was completed.
type CompletionType string

const (
	CompletionTypePending           CompletionType = "PENDING"
	CompletionTypeCompletedStandard CompletionType = "COMPLETED_STANDARD"
	CompletionTypeCompletedEmergency CompletionType = "COMPLETED_EMERGENCY"
	CompletionTypeMissed            CompletionType = "MISSED"
	CompletionTypePaused            CompletionType = "PAUSED"
)

// IsValid returns true if the CompletionType is one of the allowed values.
func (ct CompletionType) IsValid() bool {
	switch ct {
	case CompletionTypePending,
		CompletionTypeCompletedStandard,
		CompletionTypeCompletedEmergency,
		CompletionTypeMissed,
		CompletionTypePaused:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Base Model (timestamps + soft delete, without uint ID)
// ---------------------------------------------------------------------------

// BaseModel provides audit timestamps and soft-delete without overriding ID.
type BaseModel struct {
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// ---------------------------------------------------------------------------
// HabitLogKey is used for batch queries (find existing logs by composite key)
// ---------------------------------------------------------------------------

type HabitLogKey struct {
	HabitID uuid.UUID
	LogDate time.Time
}

// ---------------------------------------------------------------------------
// Domain Models
// ---------------------------------------------------------------------------

type User struct {
	ID                uuid.UUID  `json:"id"                 gorm:"type:uuid;primaryKey"`
	BaseModel
	Email             string     `json:"email"              gorm:"uniqueIndex;not null"`
	PasswordHash      string     `json:"-"                  gorm:"not null"`                          // never expose in JSON
	IdentityStatement string     `json:"identity_statement" gorm:"not null;default:''"`
	IdentityScore     float64    `json:"identity_score"     gorm:"not null;default:0.0"`
	Timezone          string     `json:"timezone"           gorm:"not null;default:'UTC'"`
	StartOfDayOffset  int        `json:"start_of_day_offset" gorm:"not null;default:0"`               // minutes from midnight
	Habits            []Habit    `json:"-"                  gorm:"foreignKey:UserID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

type Habit struct {
	ID             uuid.UUID   `json:"id"              gorm:"type:uuid;primaryKey"`
	BaseModel
	UserID         uuid.UUID   `json:"user_id"         gorm:"type:uuid;not null;index"`
	StandardTitle  string      `json:"standard_title"  gorm:"not null"`
	EmergencyTitle string      `json:"emergency_title" gorm:"not null;default:''"`
	Status         HabitStatus `json:"status"          gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	HabitLogs      []HabitLog  `json:"-"               gorm:"foreignKey:HabitID"`
}

func (h *Habit) BeforeCreate(tx *gorm.DB) (err error) {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return
}

type HabitLog struct {
	ID             uuid.UUID      `json:"id"              gorm:"type:uuid;primaryKey"`
	BaseModel
	HabitID        uuid.UUID      `json:"habit_id"        gorm:"type:uuid;not null;uniqueIndex:idx_habit_log_date;index"`
	LogDate        time.Time      `json:"log_date"        gorm:"type:date;not null;uniqueIndex:idx_habit_log_date"`
	CompletionType CompletionType `json:"completion_type" gorm:"type:varchar(30);not null;default:'PENDING'"`
	IsSynced       bool           `json:"is_synced"       gorm:"not null;default:false"`
}

func (hl *HabitLog) BeforeCreate(tx *gorm.DB) (err error) {
	if hl.ID == uuid.Nil {
		hl.ID = uuid.New()
	}
	return
}
