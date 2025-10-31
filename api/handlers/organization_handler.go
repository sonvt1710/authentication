package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lee-tech/authentication/internal/constants"
	"github.com/lee-tech/authentication/internal/models"
	"github.com/lee-tech/authentication/internal/service"
	coreErrors "github.com/lee-tech/core/errors"
	coreMiddleware "github.com/lee-tech/core/middleware"
	coreServer "github.com/lee-tech/core/server"
	"github.com/lee-tech/core/utils"
)

// OrganizationHandler exposes endpoints for managing organizations, departments, and memberships.
type OrganizationHandler struct {
	organizationService   *service.OrganizationService
	authenticationService *service.AuthenticationService
	useAuthorization      bool
	authorizationBuilder  coreMiddleware.AuthorizationRequestBuilder
}

// NewOrganizationHandler constructs a new handler instance.
func NewOrganizationHandler(orgSvc *service.OrganizationService, authSvc *service.AuthenticationService, builder coreMiddleware.AuthorizationRequestBuilder, useAuthorization bool) *OrganizationHandler {
	if builder == nil {
		builder = NewAdminAuthorizationBuilder()
	}
	return &OrganizationHandler{
		organizationService:   orgSvc,
		authenticationService: authSvc,
		useAuthorization:      useAuthorization,
		authorizationBuilder:  builder,
	}
}

// RegisterRoutes wires the routes for organization management.
func (h *OrganizationHandler) RegisterRoutes(router *mux.Router) {
	if h.organizationService == nil || h.authenticationService == nil {
		return
	}

	authenticated := router.PathPrefix("/v1/organizations").Subrouter()
	authenticated.Use(coreMiddleware.AuthMiddlewareFunc(func() string {
		return h.authenticationService.JWTSecret()
	}))

	admin := authenticated.PathPrefix("/admin").Subrouter()
	if h.useAuthorization {
		admin.Use(coreMiddleware.RequireAuthorization(h.authorizationBuilder))
	} else {
		admin.Use(coreMiddleware.RequireSuperAdmin())
	}

	coreServer.Route(admin, "/", h.CreateOrganization,
		coreServer.WithMethods(http.MethodPost),
		coreServer.WithSummary("Create organization"),
		coreServer.WithTags("Organization"),
	)

	coreServer.Route(admin, "/organizations", h.ListOrganizations,
		coreServer.WithMethods(http.MethodGet),
		coreServer.WithSummary("List organizations"),
		coreServer.WithTags("Organization"),
	)

	coreServer.Route(admin, "/organizations/{organization_id}/departments", h.CreateDepartment,
		coreServer.WithMethods(http.MethodPost),
		coreServer.WithSummary("Create department"),
		coreServer.WithTags("Organization"),
	)

	coreServer.Route(admin, "/organizations/{organization_id}/departments", h.ListDepartments,
		coreServer.WithMethods(http.MethodGet),
		coreServer.WithSummary("List departments"),
		coreServer.WithTags("Organization"),
	)

	coreServer.Route(admin, "/organizations/{organization_id}/members", h.AssignUserToOrganization,
		coreServer.WithMethods(http.MethodPost),
		coreServer.WithSummary("Assign user to organization"),
		coreServer.WithTags("Organization"),
	)

	coreServer.Route(admin, "/departments/{department_id}/members", h.AssignUserToDepartment,
		coreServer.WithMethods(http.MethodPost),
		coreServer.WithSummary("Assign user to department"),
		coreServer.WithTags("Organization"),
	)

	coreServer.Route(admin, "/users/{user_id}/organizations", h.ListUserOrganizations,
		coreServer.WithMethods(http.MethodGet),
		coreServer.WithSummary("List user organizations"),
		coreServer.WithTags("Organization"),
	)

	coreServer.Route(admin, "/users/{user_id}/departments", h.ListUserDepartments,
		coreServer.WithMethods(http.MethodGet),
		coreServer.WithSummary("List user departments"),
		coreServer.WithTags("Organization"),
	)
}

func (h *OrganizationHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var payload models.CreateOrganizationInput
	if err := utils.DecodeJSON(r.Body, &payload); err != nil {
		coreErrors.BadRequest("Invalid request body").WriteHTTP(w)
		return
	}

	org, err := h.organizationService.CreateOrganization(&payload)
	if err != nil {
		coreErrors.ValidationError(err.Error()).WriteHTTP(w)
		return
	}

	utils.RespondJSON(w, http.StatusCreated, org)
}

func (h *OrganizationHandler) ListOrganizations(w http.ResponseWriter, _ *http.Request) {
	orgs, err := h.organizationService.ListOrganizations()
	if err != nil {
		coreErrors.Internal("failed to list organizations").WithInternal(err).WriteHTTP(w)
		return
	}

	utils.RespondJSON(w, http.StatusOK, orgs)
}

func (h *OrganizationHandler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	orgID, err := utils.ParseUint64(mux.Vars(r)["organization_id"])
	if err != nil {
		coreErrors.BadRequest("invalid organization id").WriteHTTP(w)
		return
	}

	var payload models.CreateDepartmentInput
	if err := utils.DecodeJSON(r.Body, &payload); err != nil {
		coreErrors.BadRequest("Invalid request body").WriteHTTP(w)
		return
	}
	payload.OrganizationID = orgID

	dept, err := h.organizationService.CreateDepartment(&payload)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrganizationNotFound):
			coreErrors.NotFound("organization").WriteHTTP(w)
		case errors.Is(err, service.ErrDepartmentNotFound):
			coreErrors.NotFound("department").WriteHTTP(w)
		default:
			coreErrors.ValidationError(err.Error()).WriteHTTP(w)
		}
		return
	}

	utils.RespondJSON(w, http.StatusCreated, dept)
}

func (h *OrganizationHandler) ListDepartments(w http.ResponseWriter, r *http.Request) {
	orgID, err := utils.ParseUint64(mux.Vars(r)["organization_id"])
	if err != nil {
		coreErrors.BadRequest("invalid organization id").WriteHTTP(w)
		return
	}

	departments, err := h.organizationService.ListDepartments(&orgID)
	if err != nil {
		coreErrors.Internal("failed to list departments").WithInternal(err).WriteHTTP(w)
		return
	}

	utils.RespondJSON(w, http.StatusOK, departments)
}

func (h *OrganizationHandler) AssignUserToOrganization(w http.ResponseWriter, r *http.Request) {
	orgID, err := utils.ParseUint64(mux.Vars(r)["organization_id"])
	if err != nil {
		coreErrors.BadRequest("invalid organization id").WriteHTTP(w)
		return
	}

	var payload struct {
		UserID    uint64                  `json:"user_id"`
		Role      models.OrganizationRole `json:"role"`
		IsPrimary bool                    `json:"is_primary"`
	}
	if err := utils.DecodeJSON(r.Body, &payload); err != nil {
		coreErrors.BadRequest("Invalid request body").WriteHTTP(w)
		return
	}

	input := &models.AssignUserOrganizationInput{
		UserID:         payload.UserID,
		OrganizationID: orgID,
		Role:           payload.Role,
		IsPrimary:      payload.IsPrimary,
	}

	membership, err := h.organizationService.AssignUserToOrganization(input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			coreErrors.NotFound("user").WriteHTTP(w)
		case errors.Is(err, service.ErrOrganizationNotFound):
			coreErrors.NotFound("organization").WriteHTTP(w)
		default:
			coreErrors.ValidationError(err.Error()).WriteHTTP(w)
		}
		return
	}

	utils.RespondJSON(w, http.StatusCreated, membership)
}

func (h *OrganizationHandler) AssignUserToDepartment(w http.ResponseWriter, r *http.Request) {
	deptID, err := utils.ParseUint64(mux.Vars(r)["department_id"])
	if err != nil {
		coreErrors.BadRequest("invalid department id").WriteHTTP(w)
		return
	}

	var payload struct {
		UserID    uint64 `json:"user_id"`
		Role      string `json:"role"`
		IsPrimary bool   `json:"is_primary"`
	}
	if err := utils.DecodeJSON(r.Body, &payload); err != nil {
		coreErrors.BadRequest("Invalid request body").WriteHTTP(w)
		return
	}

	input := &models.AssignUserDepartmentInput{
		UserID:       &payload.UserID,
		DepartmentID: &deptID,
		Role:         payload.Role,
		IsPrimary:    payload.IsPrimary,
	}

	membership, err := h.organizationService.AssignUserToDepartment(input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			coreErrors.NotFound("user").WriteHTTP(w)
		case errors.Is(err, service.ErrDepartmentNotFound):
			coreErrors.NotFound("department").WriteHTTP(w)
		default:
			coreErrors.ValidationError(err.Error()).WriteHTTP(w)
		}
		return
	}

	utils.RespondJSON(w, http.StatusCreated, membership)
}

func (h *OrganizationHandler) ListUserOrganizations(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.ParseUint64(mux.Vars(r)["user_id"])
	if err != nil {
		coreErrors.BadRequest("invalid user id").WriteHTTP(w)
		return
	}

	memberships, err := h.organizationService.ListUserOrganizations(&userID)
	if err != nil {
		coreErrors.Internal("failed to load memberships").WithInternal(err).WriteHTTP(w)
		return
	}

	utils.RespondJSON(w, http.StatusOK, memberships)
}

func (h *OrganizationHandler) ListUserDepartments(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.ParseUint64(mux.Vars(r)["user_id"])
	if err != nil {
		coreErrors.BadRequest("invalid user id").WriteHTTP(w)
		return
	}

	memberships, err := h.organizationService.ListUserDepartments(&userID)
	if err != nil {
		coreErrors.Internal("failed to load memberships").WithInternal(err).WriteHTTP(w)
		return
	}

	utils.RespondJSON(w, http.StatusOK, memberships)
}

func parseUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}

func init() {
	coreServer.RegisterHandler(func(app *coreServer.HTTPApp) error {
		orgServiceComponent, ok := app.GetComponent(constants.ComponentKey.OrganizationService)
		if !ok {
			return fmt.Errorf("component %s not found", constants.ComponentKey.OrganizationService)
		}
		orgService, ok := orgServiceComponent.(*service.OrganizationService)
		if !ok {
			return fmt.Errorf("component %s has unexpected type %T", constants.ComponentKey.OrganizationService, orgServiceComponent)
		}

		authServiceComponent, ok := app.GetComponent(constants.ComponentKey.AuthenticationService)
		if !ok {
			return fmt.Errorf("component %s not found", constants.ComponentKey.AuthenticationService)
		}
		authService, ok := authServiceComponent.(*service.AuthenticationService)
		if !ok {
			return fmt.Errorf("component %s has unexpected type %T", constants.ComponentKey.AuthenticationService, authServiceComponent)
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

		handler := NewOrganizationHandler(orgService, authService, builder, useAuthorization)
		handler.RegisterRoutes(app.Router)
		return nil
	})
}
