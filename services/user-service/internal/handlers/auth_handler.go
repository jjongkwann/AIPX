package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"

	"user-service/internal/auth"
	"user-service/internal/models"
	"user-service/internal/repository"
)

// AuthHandler handles authentication related endpoints
type AuthHandler struct {
	userRepo         repository.UserRepository
	tokenRepo        repository.RefreshTokenRepository
	passwordHasher   auth.PasswordHasher
	jwtManager       *auth.JWTManager
	validator        *validator.Validate
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	passwordHasher auth.PasswordHasher,
	jwtManager *auth.JWTManager,
) *AuthHandler {
	return &AuthHandler{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		passwordHasher: passwordHasher,
		jwtManager:     jwtManager,
		validator:      validator.New(),
	}
}

// Signup handles user registration
// POST /api/v1/auth/signup
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		respondError(w, http.StatusBadRequest, "Weak password", err.Error())
		return
	}

	// Check if user already exists
	_, err := h.userRepo.GetUserByEmail(r.Context(), req.Email)
	if err == nil {
		respondError(w, http.StatusConflict, "User already exists", "email already registered")
		return
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to check user existence")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Hash password
	passwordHash, err := h.passwordHasher.Hash(req.Password)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Create user
	user := &models.User{
		Email:         req.Email,
		PasswordHash:  passwordHash,
		Name:          req.Name,
		IsActive:      true,
		EmailVerified: false,
	}

	if err := h.userRepo.CreateUser(r.Context(), user); err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			respondError(w, http.StatusConflict, "User already exists")
			return
		}
		log.Error().Err(err).Msg("Failed to create user")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Generate tokens
	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	refreshToken, refreshTokenHash, err := h.jwtManager.GenerateRefreshToken()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Store refresh token
	token := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(h.jwtManager.GetRefreshTTL()),
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
	}

	if err := h.tokenRepo.CreateRefreshToken(r.Context(), token); err != nil {
		log.Error().Err(err).Msg("Failed to store refresh token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Prepare response
	authResponse := &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.jwtManager.GetAccessTTLSeconds(),
		User:         user.ToResponse(),
	}

	log.Info().Str("user_id", user.ID).Str("email", user.Email).Msg("User registered successfully")
	respondSuccess(w, http.StatusCreated, authResponse, "User registered successfully")
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Get user by email
	user, err := h.userRepo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to get user")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Check if user is active
	if !user.IsActive {
		respondError(w, http.StatusUnauthorized, "Account is inactive")
		return
	}

	// Verify password
	if err := h.passwordHasher.Verify(req.Password, user.PasswordHash); err != nil {
		if errors.Is(err, auth.ErrMismatchedHashAndPassword) {
			respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to verify password")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Generate tokens
	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	refreshToken, refreshTokenHash, err := h.jwtManager.GenerateRefreshToken()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Store refresh token
	token := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(h.jwtManager.GetRefreshTTL()),
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
	}

	if err := h.tokenRepo.CreateRefreshToken(r.Context(), token); err != nil {
		log.Error().Err(err).Msg("Failed to store refresh token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Prepare response
	authResponse := &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.jwtManager.GetAccessTTLSeconds(),
		User:         user.ToResponse(),
	}

	log.Info().Str("user_id", user.ID).Str("email", user.Email).Msg("User logged in successfully")
	respondSuccess(w, http.StatusOK, authResponse, "Login successful")
}

// Refresh handles access token refresh
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		respondError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Hash the refresh token
	tokenHash := h.jwtManager.HashRefreshToken(req.RefreshToken)

	// Get refresh token from database
	token, err := h.tokenRepo.GetRefreshTokenByHash(r.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) ||
			errors.Is(err, repository.ErrRefreshTokenExpired) ||
			errors.Is(err, repository.ErrRefreshTokenRevoked) {
			respondError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
			return
		}
		log.Error().Err(err).Msg("Failed to get refresh token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Get user
	user, err := h.userRepo.GetUserByID(r.Context(), token.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusUnauthorized, "User not found")
			return
		}
		log.Error().Err(err).Str("user_id", token.UserID).Msg("Failed to get user")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Generate new access token
	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Prepare response
	refreshResponse := &models.TokenRefreshResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   h.jwtManager.GetAccessTTLSeconds(),
	}

	log.Info().Str("user_id", user.ID).Msg("Access token refreshed")
	respondSuccess(w, http.StatusOK, refreshResponse, "Token refreshed successfully")
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Hash the refresh token
	tokenHash := h.jwtManager.HashRefreshToken(req.RefreshToken)

	// Get refresh token from database
	token, err := h.tokenRepo.GetRefreshTokenByHash(r.Context(), tokenHash)
	if err != nil {
		// If token not found, consider logout successful
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			respondSuccess(w, http.StatusOK, nil, "Logout successful")
			return
		}
		log.Error().Err(err).Msg("Failed to get refresh token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Revoke the token
	if err := h.tokenRepo.RevokeToken(r.Context(), token.ID); err != nil {
		log.Error().Err(err).Str("token_id", token.ID).Msg("Failed to revoke token")
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	log.Info().Str("user_id", token.UserID).Msg("User logged out successfully")
	respondSuccess(w, http.StatusOK, nil, "Logout successful")
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
