package handlers

import (
	"errors"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"zenith/internal/middleware"
	"zenith/internal/models"
	"zenith/internal/repository"
	"zenith/internal/response"
)

// ---------------------------------------------------------------------------
// Request / Response DTOs
// ---------------------------------------------------------------------------

type RegisterRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  UserPublic `json:"user"`
}

// UserPublic is the safe, password-less representation of a user returned to clients.
type UserPublic struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	IdentityScore float64 `json:"identity_score"`
	Timezone      string  `json:"timezone"`
}

func toUserPublic(u *models.User) UserPublic {
	return UserPublic{
		ID:            u.ID.String(),
		Email:         u.Email,
		IdentityScore: u.IdentityScore,
		Timezone:      u.Timezone,
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// Register handles POST /api/v1/auth/register
// @Summary      Register a new user
// @Description  Create a new user account with email and password. Email is normalized to lowercase.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Registration Details"
// @Success      201      {object}  response.Response{data=AuthResponse}
// @Failure      400      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /auth/register [post]
func Register(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "VALIDATION_ERROR", err.Error())
			return
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))

		// Check if email is already registered
		_, err := userRepo.FindByEmail(c.Request.Context(), req.Email)
		if err == nil {
			response.Conflict(c, "An account with this email already exists")
			return
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("register: FindByEmail error: %v", err)
			response.InternalError(c)
			return
		}

		// Hash password with bcrypt
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("register: bcrypt error: %v", err)
			response.InternalError(c)
			return
		}

		user := &models.User{
			Email:        req.Email,
			PasswordHash: string(hash),
			Timezone:     "UTC",
		}

		if err := userRepo.Create(c.Request.Context(), user); err != nil {
			log.Printf("register: Create user error: %v", err)
			response.InternalError(c)
			return
		}

		token, err := middleware.GenerateToken(user.ID)
		if err != nil {
			log.Printf("register: GenerateToken error: %v", err)
			response.InternalError(c)
			return
		}

		response.Created(c, AuthResponse{
			Token: token,
			User:  toUserPublic(user),
		})
	}
}

// Login handles POST /api/v1/auth/login
// @Summary      Login user
// @Description  Authenticate user and return a JWT token for protected routes.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Login Credentials"
// @Success      200      {object}  response.Response{data=AuthResponse}
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /auth/login [post]
func Login(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "VALIDATION_ERROR", err.Error())
			return
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))

		user, err := userRepo.FindByEmail(c.Request.Context(), req.Email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Use generic message to prevent user enumeration
				response.Unauthorized(c, "Invalid email or password")
				return
			}
			log.Printf("login: FindByEmail error: %v", err)
			response.InternalError(c)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			response.Unauthorized(c, "Invalid email or password")
			return
		}

		token, err := middleware.GenerateToken(user.ID)
		if err != nil {
			log.Printf("login: GenerateToken error: %v", err)
			response.InternalError(c)
			return
		}

		response.OK(c, AuthResponse{
			Token: token,
			User:  toUserPublic(user),
		})
	}
}
