package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	coreMiddleware "github.com/lee-tech/core/middleware"
)

type adminBuilderConfig struct {
	BasePath  string
	Namespace string
	Overrides map[string]coreMiddleware.AuthorizationRequestBuilder
}

// AdminBuilderOption allows callers to customise the admin authorization builder.
type AdminBuilderOption func(*adminBuilderConfig)

// WithAdminAuthorizationBasePath overrides the path prefix that should be stripped before deriving the slug.
func WithAdminAuthorizationBasePath(basePath string) AdminBuilderOption {
	return func(cfg *adminBuilderConfig) {
		cfg.BasePath = strings.TrimSpace(basePath)
	}
}

// WithAdminAuthorizationNamespace overrides the namespace prefix for generated actions/resources.
func WithAdminAuthorizationNamespace(namespace string) AdminBuilderOption {
	return func(cfg *adminBuilderConfig) {
		cfg.Namespace = strings.TrimSpace(namespace)
	}
}

// WithAdminAuthorizationOverrides registers per-route builder overrides keyed by route name or template.
func WithAdminAuthorizationOverrides(overrides map[string]coreMiddleware.AuthorizationRequestBuilder) AdminBuilderOption {
	return func(cfg *adminBuilderConfig) {
		if len(overrides) == 0 {
			return
		}
		cfg.Overrides = make(map[string]coreMiddleware.AuthorizationRequestBuilder, len(overrides))
		for key, builder := range overrides {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" || builder == nil {
				continue
			}
			cfg.Overrides[trimmed] = builder
		}
	}
}

// NewAdminAuthorizationBuilder returns a builder that turns admin routes into authorization requests.
func NewAdminAuthorizationBuilder(opts ...AdminBuilderOption) coreMiddleware.AuthorizationRequestBuilder {
	cfg := adminBuilderConfig{
		BasePath:  "/api/v1/authentication",
		Namespace: "authentication",
		Overrides: map[string]coreMiddleware.AuthorizationRequestBuilder{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.BasePath == "" {
		cfg.BasePath = "/api/v1/authentication"
	}
	if !strings.HasPrefix(cfg.BasePath, "/") {
		cfg.BasePath = "/" + cfg.BasePath
	}
	cfg.BasePath = strings.TrimRight(cfg.BasePath, "/")
	if cfg.Namespace == "" {
		cfg.Namespace = "authentication"
	}

	return func(r *http.Request, user *coreMiddleware.AuthContext) (*coreMiddleware.AuthorizationRequest, error) {
		if r == nil {
			return nil, fmt.Errorf("http request is required")
		}

		if override := resolveOverride(r, cfg.Overrides); override != nil {
			return override(r, user)
		}

		path := deriveRoutePath(r)
		path = trimBasePath(path, cfg.BasePath)

		segments := tokeniseSegments(path)
		if len(segments) == 0 {
			segments = []string{"admin"}
		}
		slug := strings.Join(segments, ".")
		action := fmt.Sprintf("%s.%s.%s", cfg.Namespace, slug, strings.ToLower(r.Method))

		req := &coreMiddleware.AuthorizationRequest{
			Action: action,
			Resource: coreMiddleware.AuthorizationResource{
				Type: fmt.Sprintf("%s:%s", cfg.Namespace, slug),
			},
		}

		switch strings.ToLower(r.URL.Query().Get("trace")) {
		case "1", "true", "yes":
			req.Trace = true
		}

		return req, nil
	}
}

func resolveOverride(r *http.Request, overrides map[string]coreMiddleware.AuthorizationRequestBuilder) coreMiddleware.AuthorizationRequestBuilder {
	if len(overrides) == 0 || r == nil {
		return nil
	}

	current := mux.CurrentRoute(r)
	if current == nil {
		return nil
	}

	if name := current.GetName(); name != "" {
		if override, ok := overrides[name]; ok {
			return override
		}
	}

	if template, err := current.GetPathTemplate(); err == nil && template != "" {
		if override, ok := overrides[template]; ok {
			return override
		}
	}

	return nil
}

func deriveRoutePath(r *http.Request) string {
	if r == nil {
		return ""
	}

	path := r.URL.Path
	if route := mux.CurrentRoute(r); route != nil {
		if template, err := route.GetPathTemplate(); err == nil && template != "" {
			path = template
		}
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func trimBasePath(path, base string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	base = strings.TrimSpace(base)
	if base == "" {
		return strings.TrimPrefix(path, "/")
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/")

	if strings.HasPrefix(path, base) {
		path = strings.TrimPrefix(path, base)
	}

	return strings.Trim(path, "/")
}

func tokeniseSegments(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	parts := strings.Split(path, "/")
	segments := make([]string, 0, len(parts))

	for _, segment := range parts {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			continue
		}
		segment = strings.ReplaceAll(segment, "-", "_")
		segments = append(segments, segment)
	}

	return segments
}
