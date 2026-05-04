package handlers

import (
	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"zenith/internal/middleware"
	"zenith/internal/models"
	"zenith/internal/repository"
	"zenith/internal/response"
)

// Request/Response Structs
type CreateHabitRequest struct {
	StandardTitle  string `json:"standard_title" binding:"required" example:"Read 30 mins"`
	EmergencyTitle string `json:"emergency_title" binding:"required" example:"Read 5 mins"`
}

type UpdateHabitRequest struct {
	StandardTitle  string             `json:"standard_title" example:"Read 60 mins"`
	EmergencyTitle string             `json:"emergency_title" example:"Read 10 mins"`
	Status         models.HabitStatus `json:"status" example:"ACTIVE"`
}

type HabitResponse struct {
	ID             uuid.UUID          `json:"id"`
	StandardTitle  string             `json:"standard_title"`
	EmergencyTitle string             `json:"emergency_title"`
	Status         models.HabitStatus `json:"status"`
	CreatedAt      time.Time          `json:"created_at"`
}

func toHabitResponse(h *models.Habit) HabitResponse {
	return HabitResponse{
		ID:             h.ID,
		StandardTitle:  h.StandardTitle,
		EmergencyTitle: h.EmergencyTitle,
		Status:         h.Status,
		CreatedAt:      h.CreatedAt,
	}
}

// CreateHabit handles POST /api/v1/habits
// @Summary      Create a new habit
// @Description  Define a new habit with standard and emergency goals.
// @Tags         habits
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      CreateHabitRequest  true  "Habit Details"
// @Success      201      {object}  response.Response{data=HabitResponse}
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Router       /habits [post]
func CreateHabit(habitRepo repository.HabitRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateHabitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "VALIDATION_ERROR", err.Error())
			return
		}

		userID := c.MustGet(middleware.ContextKeyUserID).(uuid.UUID)

		habit := &models.Habit{
			ID:             uuid.New(),
			UserID:         userID,
			StandardTitle:  req.StandardTitle,
			EmergencyTitle: req.EmergencyTitle,
			Status:         models.HabitStatusActive,
		}

		if err := habitRepo.Create(c.Request.Context(), habit); err != nil {
			log.Printf("CreateHabit: error creating habit: %v", err)
			response.InternalError(c)
			return
		}

		response.Created(c, toHabitResponse(habit))
	}
}

// ListHabits handles GET /api/v1/habits
// @Summary      List all habits
// @Description  Retrieve all active habits for the authenticated user.
// @Tags         habits
// @Produce      json
// @Security     BearerAuth
// @Success      200      {object}  response.Response{data=[]HabitResponse}
// @Failure      401      {object}  response.Response
// @Router       /habits [get]
func ListHabits(habitRepo repository.HabitRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet(middleware.ContextKeyUserID).(uuid.UUID)

		habits, err := habitRepo.FindAllByUserID(c.Request.Context(), userID)
		if err != nil {
			log.Printf("ListHabits: error fetching habits: %v", err)
			response.InternalError(c)
			return
		}

		res := make([]HabitResponse, len(habits))
		for i, h := range habits {
			res[i] = toHabitResponse(&h)
		}

		response.OK(c, res)
	}
}

// GetHabit handles GET /api/v1/habits/:id
// @Summary      Get habit detail
// @Description  Retrieve a specific habit by its ID.
// @Tags         habits
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Habit ID (UUID)"
// @Success      200  {object}  response.Response{data=HabitResponse}
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /habits/{id} [get]
func GetHabit(habitRepo repository.HabitRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "INVALID_ID", "Invalid habit ID format")
			return
		}

		userID := c.MustGet(middleware.ContextKeyUserID).(uuid.UUID)

		habit, err := habitRepo.FindByID(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.NotFound(c, "Habit not found")
				return
			}
			log.Printf("GetHabit: error fetching habit: %v", err)
			response.InternalError(c)
			return
		}

		if habit.UserID != userID {
			response.Forbidden(c, "You do not have permission to view this habit")
			return
		}

		response.OK(c, toHabitResponse(habit))
	}
}

// UpdateHabit handles PUT /api/v1/habits/:id
// @Summary      Update a habit
// @Description  Modify the titles or status of an existing habit.
// @Tags         habits
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string              true  "Habit ID (UUID)"
// @Param        request  body      UpdateHabitRequest  true  "Updated Habit Details"
// @Success      200      {object}  response.Response{data=HabitResponse}
// @Failure      400      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Router       /habits/{id} [put]
func UpdateHabit(habitRepo repository.HabitRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "INVALID_ID", "Invalid habit ID format")
			return
		}

		var req UpdateHabitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "VALIDATION_ERROR", err.Error())
			return
		}

		userID := c.MustGet(middleware.ContextKeyUserID).(uuid.UUID)

		habit, err := habitRepo.FindByID(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.NotFound(c, "Habit not found")
				return
			}
			log.Printf("UpdateHabit: error fetching habit: %v", err)
			response.InternalError(c)
			return
		}

		if habit.UserID != userID {
			response.Forbidden(c, "You do not have permission to update this habit")
			return
		}

		// Update fields if provided
		if req.StandardTitle != "" {
			habit.StandardTitle = req.StandardTitle
		}
		if req.EmergencyTitle != "" {
			habit.EmergencyTitle = req.EmergencyTitle
		}
		if req.Status != "" {
			habit.Status = req.Status
		}

		if err := habitRepo.Update(c.Request.Context(), habit); err != nil {
			log.Printf("UpdateHabit: error updating habit: %v", err)
			response.InternalError(c)
			return
		}

		response.OK(c, toHabitResponse(habit))
	}
}

// DeleteHabit handles DELETE /api/v1/habits/:id
// @Summary      Delete a habit
// @Description  Soft delete a habit.
// @Tags         habits
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Habit ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /habits/{id} [delete]
func DeleteHabit(habitRepo repository.HabitRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "INVALID_ID", "Invalid habit ID format")
			return
		}

		userID := c.MustGet(middleware.ContextKeyUserID).(uuid.UUID)

		habit, err := habitRepo.FindByID(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.NotFound(c, "Habit not found")
				return
			}
			log.Printf("DeleteHabit: error fetching habit: %v", err)
			response.InternalError(c)
			return
		}

		if habit.UserID != userID {
			response.Forbidden(c, "You do not have permission to delete this habit")
			return
		}

		if err := habitRepo.Delete(c.Request.Context(), id); err != nil {
			log.Printf("DeleteHabit: error deleting habit: %v", err)
			response.InternalError(c)
			return
		}

		response.OK(c, gin.H{"message": "Habit deleted successfully"})
	}
}
