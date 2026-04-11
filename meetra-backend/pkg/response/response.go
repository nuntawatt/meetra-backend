// Package response provides a standard JSON response envelope used by all HTTP handlers.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// envelope is the standard JSON shape for every API response.
type envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// OK sends a 200 response with a data payload.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, envelope{Success: true, Data: data})
}

// Created sends a 201 response with a data payload.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, envelope{Success: true, Data: data})
}

// NoContent sends a 204 response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest sends a 400 error response.
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, envelope{Success: false, Error: msg})
}

// Unauthorized sends a 401 error response.
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, envelope{Success: false, Error: msg})
}

// Forbidden sends a 403 error response.
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, envelope{Success: false, Error: msg})
}

// NotFound sends a 404 error response.
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, envelope{Success: false, Error: msg})
}

// Conflict sends a 409 error response (e.g. duplicate email).
func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, envelope{Success: false, Error: msg})
}

// TooManyRequests sends a 429 error response.
func TooManyRequests(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, envelope{Success: false, Error: "rate limit exceeded"})
}

// InternalError sends a 500 error response.
func InternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, envelope{Success: false, Error: msg})
}
