package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/oapi-codegen/runtime/types"
	api "github.com/tennex/pkg/api/gen" // Generated API types
	db "github.com/tennex/pkg/db/gen"   // Generated DB types and functions
	"github.com/tennex/shared/auth"
)

// Error constants
var (
	ErrMissingToken = errors.New("missing or malformed token")
	ErrInvalidToken = errors.New("invalid token")
)

// AuthHandler handles authentication requests using generated types
type AuthHandler struct {
	queries   *db.Queries
	jwtConfig *auth.JWTConfig
	logger    *zap.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(queries *db.Queries, jwtSecret string, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		queries:   queries,
		jwtConfig: auth.DefaultJWTConfig(jwtSecret),
		logger:    logger.Named("auth_handler"),
	}
}

// Routes returns the authentication routes
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.RegisterUser)
	r.Post("/login", h.LoginUser)
	r.Get("/me", h.GetCurrentUser)

	return r
}

// RegisterUser handles user registration using generated types
func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	// Use generated API type for request validation
	var req api.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Validate required fields (OpenAPI validation happens automatically)
	if len(req.Password) < 8 {
		h.writeError(w, http.StatusBadRequest, "Password must be at least 8 characters", nil)
		return
	}

	// Check if username already exists using generated DB function
	usernameExists, err := h.queries.CheckUsernameExists(r.Context(), req.Username)
	if err != nil {
		h.logger.Error("Failed to check username", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Database error", err)
		return
	}
	if usernameExists {
		h.writeError(w, http.StatusBadRequest, "Username already exists", nil)
		return
	}

	// Check if email already exists using generated DB function
	emailExists, err := h.queries.CheckEmailExists(r.Context(), string(req.Email))
	if err != nil {
		h.logger.Error("Failed to check email", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Database error", err)
		return
	}
	if emailExists {
		h.writeError(w, http.StatusBadRequest, "Email already exists", nil)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("Failed to hash password", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to process password", err)
		return
	}

	// Create user using generated DB function and types
	createParams := db.CreateUserParams{
		Username:     req.Username,
		Email:        string(req.Email),
		PasswordHash: string(hashedPassword),
	}

	// Handle optional full_name
	if req.FullName != nil {
		createParams.FullName = pgtype.Text{String: *req.FullName, Valid: true}
	}

	user, err := h.queries.CreateUser(r.Context(), createParams)
	if err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to create user", err)
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.generateJWT(user.ID)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	// Return using generated API type
	var fullName *string
	if user.FullName.Valid {
		fullName = &user.FullName.String
	}

	response := api.AuthResponse{
		User: api.User{
			Id:        user.ID,
			Username:  user.Username,
			Email:     types.Email(user.Email),
			FullName:  fullName,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		Token:     token,
		ExpiresAt: expiresAt,
	}

	h.logger.Info("User registered successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username))

	h.writeJSON(w, http.StatusCreated, response)
}

// LoginUser handles user login using generated types
func (h *AuthHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	// Use generated API type for request validation
	var req api.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Find user by username or email using generated DB function
	user, err := h.queries.GetUserByUsernameOrEmail(r.Context(), req.Username)
	if err != nil {
		h.logger.Debug("User not found", zap.String("username", req.Username))
		h.writeError(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.logger.Debug("Invalid password", zap.String("user_id", user.ID.String()))
		h.writeError(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.generateJWT(user.ID)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	// Return using generated API type
	var fullName *string
	if user.FullName.Valid {
		fullName = &user.FullName.String
	}

	response := api.AuthResponse{
		User: api.User{
			Id:        user.ID,
			Username:  user.Username,
			Email:     types.Email(user.Email),
			FullName:  fullName,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		Token:     token,
		ExpiresAt: expiresAt,
	}

	h.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username))

	h.writeJSON(w, http.StatusOK, response)
}

// GetCurrentUser handles getting current user info using generated types
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from JWT token (middleware would normally do this)
	userID, err := h.extractUserFromToken(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}

	// Get user using generated DB function
	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err))
		h.writeError(w, http.StatusNotFound, "User not found", err)
		return
	}

	// Return using generated API type
	var fullName *string
	if user.FullName.Valid {
		fullName = &user.FullName.String
	}

	response := api.User{
		Id:        user.ID,
		Username:  user.Username,
		Email:     types.Email(user.Email),
		FullName:  fullName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *AuthHandler) generateJWT(userID uuid.UUID) (string, time.Time, error) {
	return h.jwtConfig.GenerateToken(userID)
}

func (h *AuthHandler) extractUserFromToken(r *http.Request) (uuid.UUID, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	tokenString, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		return uuid.Nil, ErrMissingToken
	}

	// Validate token using shared auth package
	claims, err := h.jwtConfig.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return claims.UserID, nil
}

func (h *AuthHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *AuthHandler) writeError(w http.ResponseWriter, status int, message string, err error) {
	h.logger.Error("API error",
		zap.String("message", message),
		zap.Error(err),
		zap.Int("status", status))

	// Use generated API type for error response
	response := api.ErrorResponse{
		Error:     message,
		Timestamp: time.Now().UTC(),
	}

	if err != nil {
		details := map[string]interface{}{"details": err.Error()}
		response.Details = &details
	}

	h.writeJSON(w, status, response)
}
