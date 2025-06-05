// handler.go - HTTP handlers layer
// This layer handles HTTP requests and responses
package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests
type Handler struct {
	service Service
}

// NewHandler creates a new handler instance
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	// Bind JSON request to struct with validation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Call service to register user
	user, err := h.service.Register(&req)
	if err != nil {
		// Handle different types of errors
		if err.Error() == "user with email "+req.Email+" already exists" {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "user_exists",
				Message: "A user with this email already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "registration_failed",
			Message: "Failed to register user",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    user,
		Message: "User registered successfully",
	})
}

// Login handles user authentication
// POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Call service to authenticate user
	token, err := h.service.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "authentication_failed",
			Message: "Invalid email or password",
		})
		return
	}

	// Return token
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: gin.H{
			"token": token,
			"type":  "Bearer",
		},
		Message: "Login successful",
	})
}

// GetUsers handles getting all users with pagination
// GET /api/v1/users?page=1&limit=10
func (h *Handler) GetUsers(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Call service to get users
	result, err := h.service.GetUsers(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch users",
		})
		return
	}

	// Return users
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    result,
	})
}

// GetUser handles getting a single user by ID
// GET /api/v1/users/:id
func (h *Handler) GetUser(c *gin.Context) {
	// Parse user ID from URL parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "User ID must be a valid number",
		})
		return
	}

	// Get current user ID from context (set by auth middleware)
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Optional: Check if user is trying to access their own profile or is admin
	// For this example, we'll allow users to view any profile
	// In a real app, you might want to restrict this

	// Call service to get user
	user, err := h.service.GetUser(id)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "user_not_found",
				Message: "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch user",
		})
		return
	}

	// Add additional info if user is viewing their own profile
	if currentUserID == id {
		// You could add additional fields here that only the user should see
		// For example: email verification status, private settings, etc.
	}

	// Return user
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    user,
	})
}

// UpdateUser handles updating user information
// PUT /api/v1/users/:id
func (h *Handler) UpdateUser(c *gin.Context) {
	// Parse user ID from URL parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "User ID must be a valid number",
		})
		return
	}

	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Check if user is trying to update their own profile
	// In a real app, you might have admin roles that can update any user
	if currentUserID != id {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only update your own profile",
		})
		return
	}

	// Bind and validate request
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	// Call service to update user
	user, err := h.service.UpdateUser(id, &req)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "user_not_found",
				Message: "User not found",
			})
			return
		}

		if err.Error() == "no updates provided" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "no_updates",
				Message: "No updates provided",
			})
			return
		}

		// Check for email already taken error
		if err.Error()[:5] == "email" && err.Error()[len(err.Error())-17:] == "is already taken" {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "email_taken",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to update user",
		})
		return
	}

	// Return updated user
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    user,
		Message: "User updated successfully",
	})
}

// DeleteUser handles deleting a user account
// DELETE /api/v1/users/:id
func (h *Handler) DeleteUser(c *gin.Context) {
	// Parse user ID from URL parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "User ID must be a valid number",
		})
		return
	}

	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Check if user is trying to delete their own account
	// In a real app, you might have admin roles that can delete any user
	if currentUserID != id {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only delete your own account",
		})
		return
	}

	// Call service to delete user
	if err := h.service.DeleteUser(id); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "user_not_found",
				Message: "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete user",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "User deleted successfully",
	})
}

// GetUserStatistics handles getting user statistics (admin endpoint example)
// GET /api/v1/admin/stats
func (h *Handler) GetUserStatistics(c *gin.Context) {
	// In a real app, you'd check if the user has admin privileges
	// For this example, we'll allow any authenticated user

	// Call service to get statistics (this demonstrates concurrent processing)
	stats, err := h.service.GetUserStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "stats_failed",
			Message: "Failed to get user statistics",
		})
		return
	}

	// Return statistics
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    stats,
	})
}

// Example of a handler that demonstrates goroutine usage
// POST /api/v1/users/:id/process
func (h *Handler) ProcessUserData(c *gin.Context) {
	// Parse user ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "User ID must be a valid number",
		})
		return
	}

	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Check authorization
	if currentUserID != id {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only process your own data",
		})
		return
	}

	// Trigger background processing
	h.service.ProcessUserAnalytics(id)

	// Return immediate response (processing happens in background)
	c.JSON(http.StatusAccepted, SuccessResponse{
		Success: true,
		Message: "User data processing started",
	})
}
