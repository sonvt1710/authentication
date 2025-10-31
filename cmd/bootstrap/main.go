package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lee-tech/authentication/config"
	"github.com/lee-tech/authentication/internal/constants"
	authservice "github.com/lee-tech/authentication/internal/service"
	coreServer "github.com/lee-tech/core/server"
)

func main() {
	orgName := flag.String("org-name", "", "Name of the bootstrap organization")
	orgDesc := flag.String("org-description", "", "Description of the bootstrap organization")
	orgDomain := flag.String("org-domain", "", "Domain/slug of the bootstrap organization")
	adminEmail := flag.String("admin-email", "", "Email address for the bootstrap admin user")
	adminUsername := flag.String("admin-username", "", "Username for the bootstrap admin user")
	adminPassword := flag.String("admin-password", "", "Password for the bootstrap admin user")
	adminFirstName := flag.String("admin-first-name", "", "First name for the bootstrap admin user")
	adminLastName := flag.String("admin-last-name", "", "Last name for the bootstrap admin user")
	forcePassword := flag.Bool("force-password", false, "Force reset of the admin password even if unchanged")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	input := &authservice.BootstrapAdminInput{
		OrganizationName:        choose(*orgName, cfg.BootstrapOrganizationName),
		OrganizationDescription: choose(*orgDesc, cfg.BootstrapOrganizationDescription),
		OrganizationDomain:      choose(*orgDomain, cfg.BootstrapOrganizationDomain),
		AdminEmail:              choose(*adminEmail, cfg.BootstrapAdminEmail),
		AdminUsername:           choose(*adminUsername, cfg.BootstrapAdminUsername),
		AdminPassword:           choose(*adminPassword, cfg.BootstrapAdminPassword),
		AdminFirstName:          choose(*adminFirstName, cfg.BootstrapAdminFirstName),
		AdminLastName:           choose(*adminLastName, cfg.BootstrapAdminLastName),
		ForcePasswordReset:      *forcePassword,
	}

	app, err := coreServer.InitializeHTTPApp(cfg.Config, &coreServer.HTTPAppOptions{
		DisableHealthRoutes: true,
	})
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}
	defer shutdownApp(app)

	svcComponent, ok := app.GetComponent(constants.ComponentKey.AuthenticationService)
	if !ok {
		log.Fatal("authentication service component not found")
	}

	authSvc, ok := svcComponent.(*authservice.AuthenticationService)
	if !ok {
		log.Fatalf("unexpected authentication service type %T", svcComponent)
	}

	org, user, err := authSvc.BootstrapAdmin(input)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	fmt.Printf("Bootstrap successful. Organization %s (%s) is active. Admin user %s (%s) ready.\n",
		org.Name, valueOrFallback(org.Domain, "n/a"), user.Email, user.Username)
}

func choose(value string, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

func valueOrFallback(value string, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func shutdownApp(app *coreServer.HTTPApp) {
	if app == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		log.Printf("warning: failed to shutdown app cleanly: %v", err)
	}
}
