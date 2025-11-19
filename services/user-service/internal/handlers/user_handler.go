package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"

	"user-service/internal/middleware"
	"user-service/internal/models"
	"user-service/internal/repository"
)

// UserHandler handles user management endpoints
type UserHandler struct {
	userRepo  repository.UserRepository
	validator *validator.Validate
}

// NewUserHandler creates a new user handler
func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo:  userRepo,
		validator: validator.New(),
	}
}

// GetMe retrieves the current user's profile
// GET /api/v1/users/me
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get user from database
	user, err := h.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondSuccess(w, http.StatusOK, user.ToResponse(), "User retrieved successfully")
}

// UpdateMe updates the current user's profile
// PATCH /api/v1/users/me
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Get current user
	user, err := h.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Update fields if provided
	updated := false
	if req.Name != nil {
		user.Name = *req.Name
		updated = true
	}
	if req.Email != nil {
		// Check if new email is already in use
		existingUser, err := h.userRepo.GetUserByEmail(r.Context(), *req.Email)
		if err == nil && existingUser.ID != userID {
			respondError(w, http.StatusConflict, "Email already in use")
			return
		}
		if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
			log.Error().Err(err).Str("email", *req.Email).Msg("Failed to check email")
			respondError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		user.Email = *req.Email
		user.EmailVerified = false // Reset email verification when email changes
		updated = true
	}

	// Only update if there are changes
	if !updated {
		respondSuccess(w, http.StatusOK, user.ToResponse(), "No changes to update")
		return
	}

	// Update user in database
	if err := h.userRepo.UpdateUser(r.Context(), user); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to update user")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	log.Info().Str("user_id", userID).Msg("User updated successfully")
	respondSuccess(w, http.StatusOK, user.ToResponse(), "User updated successfully")
}

// DeleteMe deletes (soft delete) the current user's account
// DELETE /api/v1/users/me
func (h *UserHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Soft delete user
	if err := h.userRepo.DeleteUser(r.Context(), userID); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to delete user")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	log.Info().Str("user_id", userID).Msg("User account deleted")
	respondSuccess(w, http.StatusOK, nil, "Account deleted successfully")
}
