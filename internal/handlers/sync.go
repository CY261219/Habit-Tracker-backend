package handlers

import (
	"fmt"
	"log"
<<<<<<< HEAD
=======
	"net/http"
>>>>>>> 978d64477cb29bd85011e3eb63b7a711ee7f9dde
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
<<<<<<< HEAD

	"zenith/internal/middleware"
	"zenith/internal/models"
	"zenith/internal/repository"
	"zenith/internal/response"
	"zenith/internal/services"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const maxMutationsPerRequest = 500

// ---------------------------------------------------------------------------
// Request / Response DTOs
// ---------------------------------------------------------------------------

type SyncPushRequest struct {
	Mutations []HabitLogMutation `json:"mutations" binding:"required,min=1,max=500"`
}

type HabitLogMutation struct {
	HabitID        uuid.UUID              `json:"habit_id"        binding:"required"`
	LogDate        string                 `json:"log_date"        binding:"required"`
	CompletionType models.CompletionType  `json:"completion_type" binding:"required"`
}

type SyncPushResponse struct {
	Synced  int      `json:"synced"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// PushMutations handles POST /api/v1/sync/push
// @Summary      Sync habit mutations
// @Description  Push a batch of habit log mutations (completed/missed/etc) to the server. Requires JWT token.
// @Tags         sync
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      SyncPushRequest  true  "List of mutations to sync"
// @Success      200      {object}  response.Response{data=SyncPushResponse}
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /sync/push [post]
func PushMutations(
	habitRepo repository.HabitRepository,
	habitLogRepo repository.HabitLogRepository,
	userRepo repository.UserRepository,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Parse & validate request body
		var req SyncPushRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "VALIDATION_ERROR", err.Error())
			return
		}

		// 2. Get authenticated user from JWT context
		userID, ok := c.MustGet(middleware.ContextKeyUserID).(uuid.UUID)
		if !ok {
			response.Unauthorized(c, "User identity could not be determined")
			return
		}

		// 3. Pre-validate each mutation before any DB work
		var validationErrors []string
		for i, m := range req.Mutations {
			// Validate CompletionType is an allowed value
			if !m.CompletionType.IsValid() {
				validationErrors = append(validationErrors,
					fmt.Sprintf("mutation[%d]: invalid completion_type %q", i, m.CompletionType))
				continue
			}
			// Validate log_date format
			if _, err := time.Parse("2006-01-02", m.LogDate); err != nil {
				validationErrors = append(validationErrors,
					fmt.Sprintf("mutation[%d]: invalid log_date %q, expected YYYY-MM-DD", i, m.LogDate))
			}
		}
		if len(validationErrors) > 0 {
			response.BadRequest(c, "MUTATION_VALIDATION_ERROR",
				fmt.Sprintf("%d mutation(s) failed validation: %v", len(validationErrors), validationErrors))
			return
		}

		// 4. Deduplicate mutations in memory (Client-Wins: keep last entry for same habit+date)
		type mutKey = string
		dedupMap := make(map[mutKey]HabitLogMutation, len(req.Mutations))
		for _, m := range req.Mutations {
			key := fmt.Sprintf("%s_%s", m.HabitID.String(), m.LogDate)
			dedupMap[key] = m
		}
		deduped := make([]HabitLogMutation, 0, len(dedupMap))
		for _, m := range dedupMap {
			deduped = append(deduped, m)
		}

		// 5. Collect all unique habit IDs and validate ownership
		habitIDSet := make(map[uuid.UUID]struct{}, len(deduped))
		for _, m := range deduped {
			habitIDSet[m.HabitID] = struct{}{}
		}
		habitIDs := make([]uuid.UUID, 0, len(habitIDSet))
		for id := range habitIDSet {
			habitIDs = append(habitIDs, id)
		}

		habits, err := habitRepo.FindByIDs(c.Request.Context(), habitIDs)
		if err != nil {
			log.Printf("sync/push: FindByIDs error: %v", err)
			response.InternalError(c)
			return
		}

		// Build habit → user map and verify ownership
		habitUserMap := make(map[uuid.UUID]uuid.UUID, len(habits))
		for _, h := range habits {
			habitUserMap[h.ID] = h.UserID
		}

		for _, m := range deduped {
			ownerID, found := habitUserMap[m.HabitID]
			if !found {
				response.NotFound(c, fmt.Sprintf("Habit %s", m.HabitID))
				return
			}
			if ownerID != userID {
				// Do not reveal that the habit exists for another user
				response.Forbidden(c, fmt.Sprintf("You do not have permission to modify habit %s", m.HabitID))
				return
			}
		}

		// 6. Parse dates and build HabitLog slice for upsert
		keys := make([]models.HabitLogKey, 0, len(deduped))
		logs := make([]models.HabitLog, 0, len(deduped))
		for _, m := range deduped {
			parsedDate, _ := time.Parse("2006-01-02", m.LogDate) // already validated above
			keys = append(keys, models.HabitLogKey{HabitID: m.HabitID, LogDate: parsedDate})
			logs = append(logs, models.HabitLog{
				HabitID:        m.HabitID,
				LogDate:        parsedDate,
				CompletionType: m.CompletionType,
				IsSynced:       true,
			})
		}

		// 7. Fetch existing logs to correctly revert their score contribution
		existingLogs, err := habitLogRepo.FindExisting(c.Request.Context(), keys)
		if err != nil {
			log.Printf("sync/push: FindExisting error: %v", err)
			response.InternalError(c)
			return
		}

		// Build a map: "habitID_logDate" → existing CompletionType
		existingMap := make(map[string]models.CompletionType, len(existingLogs))
		for _, el := range existingLogs {
			key := fmt.Sprintf("%s_%s", el.HabitID.String(), el.LogDate.Format("2006-01-02"))
			existingMap[key] = el.CompletionType
		}

		// 8. Fetch current user and calculate new identity score
		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil {
			log.Printf("sync/push: FindByID user error: %v", err)
			response.InternalError(c)
			return
		}

		currentScore := user.IdentityScore
		for _, m := range deduped {
			key := fmt.Sprintf("%s_%s", m.HabitID.String(), m.LogDate)

			// If a log already exists, revert its previous contribution before applying the new one
			if existingType, exists := existingMap[key]; exists {
				oldModifier := services.GetModifier(existingType)
				currentScore = services.Clamp(currentScore - oldModifier)
			}

			// Apply the new completion type's modifier
			currentScore = services.UpdateIdentityScore(currentScore, m.CompletionType)
		}

		// 9. Execute upsert + score update in a single transaction
		// We use the GORM db through the repos — for the transaction we need the raw db.
		// The repos handle their own queries; we orchestrate here.
		if err := habitLogRepo.UpsertBatch(c.Request.Context(), logs); err != nil {
			log.Printf("sync/push: UpsertBatch error: %v", err)
			response.InternalError(c)
			return
		}

		if err := userRepo.UpdateScore(c.Request.Context(), userID, currentScore); err != nil {
			log.Printf("sync/push: UpdateScore error: %v", err)
			response.InternalError(c)
			return
		}

		response.OK(c, SyncPushResponse{
			Synced:  len(logs),
			Skipped: len(req.Mutations) - len(deduped), // count of deduped entries
		})
=======
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
>>>>>>> 978d64477cb29bd85011e3eb63b7a711ee7f9dde
	}
}
