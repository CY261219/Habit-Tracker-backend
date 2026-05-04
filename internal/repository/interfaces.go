package repository

import (
	"context"

	"github.com/google/uuid"

	"zenith/internal/models"
)

// UserRepository defines the contract for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateScore(ctx context.Context, id uuid.UUID, newScore float64) error
	Save(ctx context.Context, user *models.User) error
}

// HabitRepository defines the contract for habit data access.
type HabitRepository interface {
	Create(ctx context.Context, habit *models.Habit) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Habit, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Habit, error)
	FindAllByUserID(ctx context.Context, userID uuid.UUID) ([]models.Habit, error)
	Update(ctx context.Context, habit *models.Habit) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// HabitLogRepository defines the contract for habit log data access.
type HabitLogRepository interface {
	// UpsertBatch performs a bulk upsert using Client-Wins conflict resolution.
	// On conflict of (habit_id, log_date), it updates completion_type and is_synced.
	UpsertBatch(ctx context.Context, logs []models.HabitLog) error

	// FindExisting fetches existing logs that match any of the given (habit_id, log_date) pairs.
	FindExisting(ctx context.Context, keys []models.HabitLogKey) ([]models.HabitLog, error)
}
