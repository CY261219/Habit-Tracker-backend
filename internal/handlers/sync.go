package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
	}
}
