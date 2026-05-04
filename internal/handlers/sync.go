package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zenith/internal/models"
	"zenith/internal/services"
)

type SyncPushRequest struct {
	Mutations []HabitLogMutation `json:"mutations" binding:"required"`
}

type HabitLogMutation struct {
	HabitID        uuid.UUID `json:"habit_id" binding:"required"`
	LogDate        string    `json:"log_date" binding:"required"` // Format: YYYY-MM-DD
	CompletionType string    `json:"completion_type" binding:"required"`
}

// PushMutations handles the POST /api/v1/sync/push endpoint
func PushMutations(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SyncPushRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Acknowledge receipt quickly
		c.JSON(http.StatusOK, gin.H{"status": "queued", "count": len(req.Mutations)})

		// Spawn a goroutine for background processing
		go processMutations(db, req.Mutations)
	}
}

func processMutations(db *gorm.DB, mutations []HabitLogMutation) {
	if len(mutations) == 0 {
		return
	}

	// Deduplicate mutations in memory (Client-Wins: keep last)
	dedupMap := make(map[string]HabitLogMutation)
	for _, m := range mutations {
		key := fmt.Sprintf("%s_%s", m.HabitID.String(), m.LogDate)
		dedupMap[key] = m
	}

	var dedupedMutations []HabitLogMutation
	for _, m := range dedupMap {
		dedupedMutations = append(dedupedMutations, m)
	}

	// Prepare data for upsert
	var logs []models.HabitLog
	for _, m := range dedupedMutations {
		parsedDate, err := time.Parse("2006-01-02", m.LogDate)
		if err != nil {
			log.Printf("Failed to parse log_date: %s, error: %v", m.LogDate, err)
			continue // Skip invalid dates or handle differently based on requirements
		}

		logs = append(logs, models.HabitLog{
			ID:             uuid.New(), // Generated new ID in case of insert
			HabitID:        m.HabitID,
			LogDate:        parsedDate,
			CompletionType: m.CompletionType,
			IsSynced:       true, // It's from sync push
		})
	}

	if len(logs) == 0 {
		return
	}

	// Execute transaction
	err := db.Transaction(func(tx *gorm.DB) error {
		// 0. Fetch existing logs to calculate score delta correctly
		var existingLogs []models.HabitLog
		var conditions []string
		var args []interface{}

		for _, l := range logs {
			conditions = append(conditions, "(habit_id = ? AND log_date = ?)")
			args = append(args, l.HabitID, l.LogDate)
		}

		query := ""
		for i, c := range conditions {
			if i > 0 {
				query += " OR "
			}
			query += c
		}

		if err := tx.Where(query, args...).Find(&existingLogs).Error; err != nil {
			return fmt.Errorf("fetching existing logs failed: %w", err)
		}

		existingLogMap := make(map[string]string)
		for _, el := range existingLogs {
			key := fmt.Sprintf("%s_%s", el.HabitID.String(), el.LogDate.Format("2006-01-02"))
			existingLogMap[key] = el.CompletionType
		}

		// 1. Bulk Upsert "Client-Wins"
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "habit_id"}, {Name: "log_date"}},
			DoUpdates: clause.AssignmentColumns([]string{"completion_type", "is_synced"}),
		}).Create(&logs).Error; err != nil {
			return fmt.Errorf("bulk upsert failed: %w", err)
		}

		// 2. Fetch impacted Users to recalculate Identity Score
		// Optimization: Group mutations by UserID
		habitIDs := make([]uuid.UUID, 0, len(logs))
		for _, log := range logs {
			habitIDs = append(habitIDs, log.HabitID)
		}

		// Fetch habits to map them to users
		var habits []models.Habit
		if err := tx.Where("id IN ?", habitIDs).Find(&habits).Error; err != nil {
			return fmt.Errorf("fetching habits failed: %w", err)
		}

		habitUserMap := make(map[uuid.UUID]uuid.UUID)
		userIDs := make([]uuid.UUID, 0)
		for _, h := range habits {
			habitUserMap[h.ID] = h.UserID
			userIDs = append(userIDs, h.UserID)
		}

		// Fetch users and lock them FOR UPDATE
		var users []models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return fmt.Errorf("fetching users FOR UPDATE failed: %w", err)
		}

		userMap := make(map[uuid.UUID]*models.User)
		for i := range users {
			userMap[users[i].ID] = &users[i]
		}

		// Calculate new scores
		for _, m := range dedupedMutations {
			userID, ok := habitUserMap[m.HabitID]
			if !ok {
				// Habit might not exist or wasn't fetched
				continue
			}

			user, ok := userMap[userID]
			if !ok {
				continue
			}

			key := fmt.Sprintf("%s_%s", m.HabitID.String(), m.LogDate)
			existingType, exists := existingLogMap[key]

			if exists {
				// Revert the previous score adjustment
				// UpdateIdentityScore expects a current score and completion type to *add*
				// To revert, we need to subtract the modifier of the existing type.
				// For simplicity, we can calculate the modifier directly
				revertModifier := 0.0
				switch existingType {
				case services.CompletionTypeStandard:
					revertModifier = -2.0
				case services.CompletionTypeEmergency:
					revertModifier = -0.5
				case services.CompletionTypeMissed:
					revertModifier = 1.0 // subtracting a negative is adding
				}
				user.IdentityScore += revertModifier
			}

			// We apply score adjustment per mutation
			user.IdentityScore = services.UpdateIdentityScore(user.IdentityScore, m.CompletionType)
		}

		// Save updated users
		for _, u := range userMap {
			if err := tx.Save(u).Error; err != nil {
				return fmt.Errorf("saving user %v failed: %w", u.ID, err)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Background processMutations failed: %v", err)
	}
}
