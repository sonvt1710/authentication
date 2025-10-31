package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/lee-tech/authentication/internal/constants"
	"github.com/lee-tech/authentication/internal/models"
	"github.com/lee-tech/authentication/internal/service"
	coreErrors "github.com/lee-tech/core/errors"
	coreMiddleware "github.com/lee-tech/core/middleware"
	coreServer "github.com/lee-tech/core/server"
	"github.com/lee-tech/core/utils"
)

// AuthenticationHandler handles authentication endpoints
type AuthenticationHandler struct {
	authenticationService *service.AuthenticationService
	useAuthorization      bool
	authorizationBuilder  coreMiddleware.AuthorizationRequestBuilder
}

// NewAuthenticationHandler creates a new auth handler
func NewAuthenticationHandler(authService *service.AuthenticationService, useAuthorization bool, builder coreMiddleware.AuthorizationRequestBuilder) *AuthenticationHandler {
	if builder == nil {
		builder = NewAdminAuthorizationBuilder()
	}
	return &AuthenticationHandler{
		authenticationService: authService,
		useAuthorization:      useAuthorization,
		authorizationBuilder:  builder,
	}
}

// RegisterRoutes registers all auth routes
func (h *AuthenticationHandler) RegisterRoutes(router *mux.Router) {
	// Public routes (no auth required)
	coreServer.Route(router, "/v1/login",
		h.Login,
		coreServer.WithMethods(http.MethodPost),
		coreServer.WithSummary("Login"),
		coreServer.WithRequestBody(&coreServer.BodyMeta{
			Required: true,
			ModelKey: "login-request",
			Example: map[string]any{
				"username":        "root-admin",
				"password":        "ChangeMe123!",
				"organization_id": 1,
				"role_id":         1,
			},
		}),
		coreServer.WithResponseMeta(map[int]coreServer.BodyMeta{
			http.StatusOK: {
				Required:    true,
				ModelKey:    "login-response",
				Description: "Successful login response",
				Example: map[string]any{
					"access_token":        "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsi",
					"refresh_token":       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhd",
					"token_type":          "Bearer",
					"user":                map[string]any{},
					"logged_organization": map[string]any{},
				},
			},
			http.StatusForbidden: {
				IsIgnored: true,
			},
		}),
		coreServer.WithDescription("Authenticate a user and return tokens"),
		coreServer.WithTags("Authentication"),
		coreServer.AllowAnonymous(),
	)

	// Registration endpoint is disabled for now
	// coreServer.Route(router, "/v1/register", h.Register,
	// 	coreServer.WithMethods(http.MethodPost),
	// 	coreServer.WithSummary("Register"),
	// 	coreServer.WithDescription("Register a new user account"),
	// 	coreServer.WithTags("Authentication"),
	// 	coreServer.AllowAnonymous(),
	// )

	// Health check endpoint
	coreServer.Route(router, "/v1/health", h.Health,
		coreServer.WithMethods(http.MethodGet),
		coreServer.WithSummary("Authentication health"),
		coreServer.WithTags("Authentication"),
		coreServer.AllowAnonymous(),
	)

	// Protected routes (authentication required)
	authenticated := router.PathPrefix("/v1/auth").Subrouter()
	authenticated.Use(coreMiddleware.AuthMiddlewareFunc(func() string {
		return h.authenticationService.JWTSecret()
	}))

	coreServer.Route(authenticated, "/me", h.Me,
		coreServer.WithMethods(http.MethodGet),
		coreServer.WithSummary("Current user"),
		coreServer.WithDescription("Retrieve the authenticated user's profile"),
		coreServer.WithTags("Authentication"),
		coreServer.RequireAuth(),
	)

	coreServer.Route(router, "/refresh", h.RefreshToken,
		coreServer.WithMethods(http.MethodPost),
		coreServer.WithSummary("Refresh token"),
		coreServer.WithDescription("Refresh the access token using a refresh token"),
		coreServer.WithTags("Authentication"),
		coreServer.AllowAnonymous(),
	)

	// Administrative routes (require elevated permissions)
	adminRouter := authenticated.PathPrefix("/admin").Subrouter()
	if h.useAuthorization {
		adminRouter.Use(coreMiddleware.RequireAuthorization(h.authorizationBuilder))
	} else {
		adminRouter.Use(coreMiddleware.RequireSuperAdmin())
	}
	coreServer.Route(adminRouter, "/users", h.ListUsers,
		coreServer.WithMethods(http.MethodGet),
		coreServer.WithSummary("List users (admin)"),
		coreServer.WithDescription("List users with administrative privileges"),
		coreServer.WithTags("Administration"),
		coreServer.RequireAuth(),
	)
}

// Login handles user login
func (h *AuthenticationHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		coreErrors.BadRequest("Invalid request body").WriteHTTP(w)
		return
	}

	// Validate request
	if req.Username == "" || req.Password == "" {
		coreErrors.ValidationError("Username and password are required").WriteHTTP(w)
		return
	}
	if req.OrganizationID == 0 || req.RoleID == 0 {
		coreErrors.ValidationError("Organization ID and Role ID are required").WriteHTTP(w)
		return
	}

	// Authenticate user
	response, err := h.authenticationService.Login(&req)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			coreErrors.Unauthorized("Invalid username or password").WriteHTTP(w)
		case service.ErrAccountLocked:
			coreErrors.Forbidden("Account is locked due to too many failed attempts").WriteHTTP(w)
		case service.ErrAccountInactive:
			coreErrors.Forbidden("Account is not active").WriteHTTP(w)
		default:
			coreErrors.Internal("An error occurred during login").WriteHTTP(w)
		}
		return
	}

	// Return success response
	utils.RespondJSON(w, http.StatusOK, response)
}

// Register handles user registration
// func (h *AuthenticationHandler) Register(w http.ResponseWriter, r *http.Request) {
// 	var req models.RegisterRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		coreErrors.BadRequest("Invalid request body").WriteHTTP(w)
// 		return
// 	}

// 	// Basic validation
// 	if req.Email == "" || req.Username == "" || req.Password == "" {
// 		coreErrors.ValidationError("Email, username, and password are required").WriteHTTP(w)
// 		return
// 	}

// 	// Validate email format
// 	if !utils.IsEmail(req.Email) {
// 		coreErrors.ValidationError("Invalid email format").WriteHTTP(w)
// 		return
// 	}

// 	// Validate password strength
// 	if len(req.Password) < 8 {
// 		coreErrors.ValidationError("Password must be at least 8 characters long").WriteHTTP(w)
// 		return
// 	}

// 	// Register user
// 	user, err := h.authenticationService.Register(&req)
// 	if err != nil {
// 		if err.Error() == "email already registered" || err.Error() == "username already taken" {
// 			coreErrors.Conflict(err.Error()).WriteHTTP(w)
// 		} else {
// 			coreErrors.Internal("Failed to register user").WriteHTTP(w)
// 		}
// 		return
// 	}

// 	// Return user info (without password)
// 	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
// 		"message": "User registered successfully",
// 		"user":    user.ToUserInfo(),
// 	})
// }

// RefreshToken handles token refresh
func (h *AuthenticationHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		coreErrors.BadRequest("Invalid request body").WriteHTTP(w)
		return
	}

	if req.RefreshToken == "" {
		coreErrors.ValidationError("Refresh token is required").WriteHTTP(w)
		return
	}

	// Refresh tokens
	response, err := h.authenticationService.RefreshToken(req.RefreshToken)
	if err != nil {
		if err == service.ErrInvalidToken {
			coreErrors.Unauthorized("Invalid or expired refresh token").WriteHTTP(w)
		} else {
			coreErrors.Internal("Failed to refresh token").WriteHTTP(w)
		}
		return
	}

	// Return new tokens
	utils.RespondJSON(w, http.StatusOK, response)
}

// Health returns service health status
func (h *AuthenticationHandler) Health(w http.ResponseWriter, r *http.Request) {
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "auth-service",
	})
}

// Me returns details about the authenticated user.
func (h *AuthenticationHandler) Me(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value(coreMiddleware.UserIDKey)
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		coreErrors.Unauthorized("user context missing").WriteHTTP(w)
		return
	}

	userID, err := utils.ParseUint64(userIDStr)
	if err != nil {
		coreErrors.Unauthorized("invalid user identifier").WriteHTTP(w)
		return
	}

	userInfo, err := h.authenticationService.GetUserInfoByID(userID)
	if err != nil {
		coreErrors.Internal("failed to load user profile").WithInternal(err).WriteHTTP(w)
		return
	}
	if userInfo == nil {
		coreErrors.NotFound("user").WriteHTTP(w)
		return
	}

	utils.RespondJSON(w, http.StatusOK, userInfo)
}

// ListUsers returns a paginated list of users. Super admin or explicit permission required.
func (h *AuthenticationHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !coreMiddleware.HasPermission(r, "auth.users.read") {
		coreErrors.Forbidden("insufficient permissions").WriteHTTP(w)
		return
	}

	page := 1
	pageSize := 20

	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if parsed, err := strconv.Atoi(pageParam); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if sizeParam := r.URL.Query().Get("page_size"); sizeParam != "" {
		if parsed, err := strconv.Atoi(sizeParam); err == nil && parsed > 0 {
			if parsed > 100 {
				parsed = 100
			}
			pageSize = parsed
		}
	}

	offset := (page - 1) * pageSize

	userInfos, total, err := h.authenticationService.ListUsers(offset, pageSize)
	if err != nil {
		coreErrors.Internal("failed to list users").WithInternal(err).WriteHTTP(w)
		return
	}

	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"data": userInfos,
		"pagination": map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func init() {
	coreServer.RegisterHandler(func(app *coreServer.HTTPApp) error {
		serviceComponent, ok := app.GetComponent(constants.ComponentKey.AuthenticationService)
		if !ok {
			return fmt.Errorf("component %s not found", constants.ComponentKey.AuthenticationService)
		}

		authenticationService, ok := serviceComponent.(*service.AuthenticationService)
		if !ok {
			return fmt.Errorf("component %s has unexpected type %T", constants.ComponentKey.AuthenticationService, serviceComponent)
		}

		var builder coreMiddleware.AuthorizationRequestBuilder
		if builderComponent, ok := app.GetComponent(constants.ComponentKey.AdminAuthorizationBuilder); ok {
			if resolved, ok := builderComponent.(coreMiddleware.AuthorizationRequestBuilder); ok {
				builder = resolved
			}
		}

		useAuthorization := false
		if flagComponent, ok := app.GetComponent(constants.ComponentKey.AuthorizationEnabled); ok {
			if enabled, ok := flagComponent.(bool); ok {
				useAuthorization = enabled
			}
		}

		handler := NewAuthenticationHandler(authenticationService, useAuthorization, builder)
		handler.RegisterRoutes(app.Router)
		return nil
	})
}
