package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/lee-tech/authentication/internal/service"
	coreErrors "github.com/lee-tech/core/errors"
	coreServer "github.com/lee-tech/core/server"
)

// TokenIntrospectionRequest represents a token introspection request
type TokenIntrospectionRequest struct {
	Token string `json:"token" validate:"required"`
}

// TokenIntrospectionResponse represents a token introspection response
type TokenIntrospectionResponse struct {
	Active         bool     `json:"active"`
	Sub            string   `json:"sub,omitempty"`
	Username       string   `json:"username,omitempty"`
	Email          string   `json:"email,omitempty"`
	OrganizationID string   `json:"organization_id,omitempty"`
	DepartmentID   string   `json:"department_id,omitempty"`
	RoleIDs        string   `json:"role_id,omitempty"`
	Scopes         []string `json:"scope,omitempty"`
	IssuedAt       *int64   `json:"iat,omitempty"`
	ExpiresAt      *int64   `json:"exp,omitempty"`
	NotBefore      *int64   `json:"nbf,omitempty"`
	ClientID       string   `json:"client_id,omitempty"`
	TokenType      string   `json:"token_type,omitempty"`
}

// TokenIntrospectionHandler handles token introspection requests
type TokenIntrospectionHandler struct {
	authService         *service.AuthenticationService
	introspectionSecret string
}

// NewTokenIntrospectionHandler creates a new token introspection handler
func NewTokenIntrospectionHandler(authService *service.AuthenticationService, introspectionSecret string) *TokenIntrospectionHandler {
	return &TokenIntrospectionHandler{
		authService:         authService,
		introspectionSecret: introspectionSecret,
	}
}

// RegisterRoutes registers token introspection routes
func (h *TokenIntrospectionHandler) RegisterRoutes(router *mux.Router) {
	coreServer.Route(router, "/v1/token/introspect", h.Introspect,
		coreServer.WithMethods(http.MethodPost),
		coreServer.WithSummary("Token Introspection"),
		coreServer.WithDescription("Introspect an access or refresh token to validate and retrieve metadata"),
		coreServer.WithTags("Authentication"),
		coreServer.WithRequestBody(&coreServer.BodyMeta{
			Required: true,
			ModelKey: "token-introspection-request",
			Example: map[string]interface{}{
				"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			},
		}),
		coreServer.WithResponseMeta(map[int]coreServer.BodyMeta{
			http.StatusOK: {
				Required: true,
				ModelKey: "token-introspection-response",
				Example: map[string]interface{}{
					"active":     true,
					"sub":        "1234567890",
					"username":   "johndoe",
					"email":      "john@example.com",
					"token_type": "access",
					"exp":        1234567890,
					"iat":        1234567890,
				},
			},
		}),
		coreServer.AllowAnonymous(),
	)
}

// Introspect validates a token and returns its metadata
func (h *TokenIntrospectionHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	var req TokenIntrospectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		coreErrors.BadRequest("Invalid request body").WriteHTTP(w)
		return
	}

	// Parse and validate the token
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(req.Token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, coreErrors.Unauthorized("Invalid signing method")
		}
		return []byte(h.introspectionSecret), nil
	})

	response := &TokenIntrospectionResponse{
		Active: false,
	}

	if err != nil || !token.Valid {
		// Token is invalid or expired
		h.writeResponse(w, response)
		return
	}

	// Token is valid - populate response
	response.Active = true
	response.TokenType = "access"

	// Extract standard claims
	if sub, ok := claims["sub"].(string); ok {
		response.Sub = sub
	}

	if username, ok := claims["username"].(string); ok {
		response.Username = username
	}

	if email, ok := claims["email"].(string); ok {
		response.Email = email
	}

	if orgID, ok := claims["org_id"]; ok {
		if uint64Val, ok := orgID.(uint64); ok {
			response.OrganizationID = uint64ToString(uint64Val)
		} else if strVal, ok := orgID.(string); ok {
			response.OrganizationID = strVal
		}
	}

	// Extract timestamps
	if iat, ok := claims["iat"].(float64); ok {
		response.IssuedAt = int64Ptr(int64(iat))
	}

	if exp, ok := claims["exp"].(float64); ok {
		response.ExpiresAt = int64Ptr(int64(exp))
	}

	if nbf, ok := claims["nbf"].(float64); ok {
		response.NotBefore = int64Ptr(int64(nbf))
	}

	// Check if token is expired
	if response.ExpiresAt != nil && time.Now().Unix() > *response.ExpiresAt {
		response.Active = false
	}

	h.writeResponse(w, response)
}

// writeResponse writes the introspection response
func (h *TokenIntrospectionHandler) writeResponse(w http.ResponseWriter, resp *TokenIntrospectionResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		coreErrors.Internal("Failed to encode response").WriteHTTP(w)
	}
}

// Helper functions
func int64Ptr(i int64) *int64 {
	return &i
}

func uint64ToString(u uint64) string {
	// Convert uint64 to string
	return fmt.Sprintf("%d", u)
}
