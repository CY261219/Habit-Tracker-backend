package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"zenith/internal/models"
	"zenith/internal/repository"
)

type habitRepo struct {
	db *gorm.DB
}

// NewHabitRepository returns a PostgreSQL-backed HabitRepository.
func NewHabitRepository(db *gorm.DB) repository.HabitRepository {
	return &habitRepo{db: db}
}

func (r *habitRepo) Create(ctx context.Context, habit *models.Habit) error {
	if err := r.db.WithContext(ctx).Create(habit).Error; err != nil {
		return fmt.Errorf("habitRepo.Create: %w", err)
	}
	return nil
}

func (r *habitRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Habit, error) {
	var habit models.Habit
	if err := r.db.WithContext(ctx).First(&habit, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("habitRepo.FindByID: %w", err)
	}
	return &habit, nil
}

func (r *habitRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Habit, error) {
	var habits []models.Habit
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&habits).Error; err != nil {
		return nil, fmt.Errorf("habitRepo.FindByIDs: %w", err)
	}
	return habits, nil
}

func (r *habitRepo) FindAllByUserID(ctx context.Context, userID uuid.UUID) ([]models.Habit, error) {
	var habits []models.Habit
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&habits).Error; err != nil {
		return nil, fmt.Errorf("habitRepo.FindAllByUserID: %w", err)
	}
	return habits, nil
}

func (r *habitRepo) Update(ctx context.Context, habit *models.Habit) error {
	if err := r.db.WithContext(ctx).Save(habit).Error; err != nil {
		return fmt.Errorf("habitRepo.Update: %w", err)
	}
	return nil
}

func (r *habitRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Habit{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("habitRepo.Delete: %w", err)
	}
	return nil
}
