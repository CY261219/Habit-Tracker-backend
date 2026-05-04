package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email             string    `gorm:"uniqueIndex;not null"`
	IdentityStatement string
	IdentityScore     float64   `gorm:"default:0.0"`
	Timezone          string
	StartOfDayOffset  string    `gorm:"type:time without time zone;default:'00:00:00'"`
	Habits            []Habit   `gorm:"foreignKey:UserID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

type Habit struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	StandardTitle  string
	EmergencyTitle string
	Status         string     `gorm:"type:varchar(20);default:'ACTIVE'"` // 'ACTIVE', 'PAUSED', 'ARCHIVED'
	HabitLogs      []HabitLog `gorm:"foreignKey:HabitID"`
}

func (h *Habit) BeforeCreate(tx *gorm.DB) (err error) {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return
}

type HabitLog struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	HabitID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_habit_log_date"`
	LogDate        time.Time `gorm:"type:date;uniqueIndex:idx_habit_log_date"`
	CompletionType string    `gorm:"type:varchar(30);default:'PENDING'"` // 'PENDING', 'COMPLETED_STANDARD', 'COMPLETED_EMERGENCY', 'MISSED'
	IsSynced       bool      `gorm:"default:false"`
}

func (hl *HabitLog) BeforeCreate(tx *gorm.DB) (err error) {
	if hl.ID == uuid.Nil {
		hl.ID = uuid.New()
	}
	return
}
