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

// IsValid checks if the completion type is a recognized value.
func (ct CompletionType) IsValid() bool {
	switch ct {
	case CompletionTypePending, CompletionTypeCompletedStandard,
		CompletionTypeCompletedEmergency, CompletionTypeMissed, CompletionTypePaused:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Database Entities
// ---------------------------------------------------------------------------

// User represents a person using the habit tracker.
type User struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	Email         string         `gorm:"uniqueIndex;not null"`
	PasswordHash  string         `gorm:"not null"`
	IdentityScore float64        `gorm:"default:0.0"`
	Timezone      string         `gorm:"default:'UTC'"`
}

// Habit represents a recurring goal a user wants to achieve.
type Habit struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	UserID         uuid.UUID      `gorm:"type:uuid;index;not null"`
	StandardTitle  string         `gorm:"not null"`
	EmergencyTitle string         `gorm:"not null"`
	Status         HabitStatus    `gorm:"default:'ACTIVE'"`

	// Associations
	User User `gorm:"foreignKey:UserID"`
}

// HabitLog tracks the daily completion status of a habit.
type HabitLog struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	HabitID        uuid.UUID      `gorm:"type:uuid;index;not null"`
	LogDate        time.Time      `gorm:"type:date;not null"`
	CompletionType CompletionType `gorm:"not null"`
	IsSynced       bool           `gorm:"default:false"`

	// Composite Unique Constraint: one log per habit per day
	// Defined via gorm tags on specific fields or at migration level.
	// In migrations: UNIQUE(habit_id, log_date)

	// Associations
	Habit Habit `gorm:"foreignKey:HabitID"`
}

// HabitLogKey is used for batch lookups of existing logs.
type HabitLogKey struct {
	HabitID uuid.UUID
	LogDate time.Time
}
