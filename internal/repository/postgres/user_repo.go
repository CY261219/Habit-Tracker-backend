package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"zenith/internal/models"
	"zenith/internal/repository"
)

type userRepo struct {
	db *gorm.DB
}

// NewUserRepository returns a PostgreSQL-backed UserRepository.
func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}
	return &user, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	return &user, nil
}

func (r *userRepo) UpdateScore(ctx context.Context, id uuid.UUID, newScore float64) error {
	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Update("identity_score", newScore).Error; err != nil {
		return fmt.Errorf("userRepo.UpdateScore: %w", err)
	}
	return nil
}

func (r *userRepo) Save(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("userRepo.Save: %w", err)
	}
	return nil
}
