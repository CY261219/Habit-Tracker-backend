package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard API envelope returned by all endpoints.
type Response struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrDetail `json:"error,omitempty"`
}

// ErrDetail carries a machine-readable code and a human-readable message.
type ErrDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OK responds with 200 and wraps data in the standard envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

// Created responds with 201 and wraps data in the standard envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{Success: true, Data: data})
}

// NoContent responds with 204.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest responds with 400.
func BadRequest(c *gin.Context, code, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error:   &ErrDetail{Code: code, Message: message},
	})
}

// Unauthorized responds with 401.
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error:   &ErrDetail{Code: "UNAUTHORIZED", Message: message},
	})
}

// Forbidden responds with 403.
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Success: false,
		Error:   &ErrDetail{Code: "FORBIDDEN", Message: message},
	})
}

// NotFound responds with 404.
func NotFound(c *gin.Context, resource string) {
	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error:   &ErrDetail{Code: "NOT_FOUND", Message: resource + " not found"},
	})
}

// Conflict responds with 409.
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, Response{
		Success: false,
		Error:   &ErrDetail{Code: "CONFLICT", Message: message},
	})
}

// InternalError responds with 500. Details are intentionally hidden from the client.
func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error:   &ErrDetail{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred. Please try again later."},
	})
}
