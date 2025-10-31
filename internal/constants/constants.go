package constants

var ComponentKey = struct {
	AuthenticationService     string
	AuthenticationConfig      string
	AuthenticationUserRepo    string
	OrganizationRepository    string
	OrganizationService       string
	AdminAuthorizationBuilder string
	AuthorizationEnabled      string
}{
	AuthenticationService:     "authentication.service.authentication",
	AuthenticationConfig:      "config.authentication",
	AuthenticationUserRepo:    "authentication.repository.user",
	OrganizationRepository:    "authentication.repository.organization",
	OrganizationService:       "authentication.service.organization",
	AdminAuthorizationBuilder: "authentication.authorization.builder.admin",
	AuthorizationEnabled:      "authentication.authorization.enabled",
}
