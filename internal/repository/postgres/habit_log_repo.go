package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zenith/internal/models"
	"zenith/internal/repository"
)

type habitLogRepo struct {
	db *gorm.DB
}

// NewHabitLogRepository returns a PostgreSQL-backed HabitLogRepository.
func NewHabitLogRepository(db *gorm.DB) repository.HabitLogRepository {
	return &habitLogRepo{db: db}
}

// UpsertBatch performs a bulk upsert using the UNIQUE constraint on (habit_id, log_date).
// On conflict: Client-Wins — overwrites completion_type and is_synced.
func (r *habitLogRepo) UpsertBatch(ctx context.Context, logs []models.HabitLog) error {
	if len(logs) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "habit_id"}, {Name: "log_date"}},
			DoUpdates: clause.AssignmentColumns([]string{"completion_type", "is_synced", "updated_at"}),
		}).
		Create(&logs).Error
	if err != nil {
		return fmt.Errorf("habitLogRepo.UpsertBatch: %w", err)
	}
	return nil
}

// FindExisting retrieves habit logs that match any of the provided (habit_id, log_date) pairs.
// Uses a VALUES-based query approach for correctness and safety across all batch sizes.
func (r *habitLogRepo) FindExisting(ctx context.Context, keys []models.HabitLogKey) ([]models.HabitLog, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	// Build a VALUES list: (habit_id::uuid, log_date::date), ...
	// This avoids string concatenation injection risk because all values go through
	// parameterized bindings, one pair at a time.
	// For the batch sizes expected (≤500), this is fast and safe.

	// Collect unique habit_ids for a pre-filter (uses index on habit_id)
	habitIDSet := make(map[uuid.UUID]struct{}, len(keys))
	for _, k := range keys {
		habitIDSet[k.HabitID] = struct{}{}
	}
	habitIDs := make([]uuid.UUID, 0, len(habitIDSet))
	for id := range habitIDSet {
		habitIDs = append(habitIDs, id)
	}

	// Fetch all logs for those habit_ids (small result set), then filter in Go
	var candidateLogs []models.HabitLog
	if err := r.db.WithContext(ctx).
		Where("habit_id IN ? AND deleted_at IS NULL", habitIDs).
		Find(&candidateLogs).Error; err != nil {
		return nil, fmt.Errorf("habitLogRepo.FindExisting: %w", err)
	}

	// Build lookup set of requested keys
	type key struct {
		HabitID uuid.UUID
		LogDate string
	}
	keySet := make(map[key]struct{}, len(keys))
	for _, k := range keys {
		keySet[key{k.HabitID, k.LogDate.Format("2006-01-02")}] = struct{}{}
	}

	// Filter candidates to only those matching the exact (habit_id, log_date) pairs
	result := make([]models.HabitLog, 0)
	for _, log := range candidateLogs {
		k := key{log.HabitID, log.LogDate.Format("2006-01-02")}
		if _, ok := keySet[k]; ok {
			result = append(result, log)
		}
	}

	return result, nil
}
