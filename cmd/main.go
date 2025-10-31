package main

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/lee-tech/authentication/api/handlers"
	"github.com/lee-tech/authentication/config"
	"github.com/lee-tech/authentication/internal/constants"
	"github.com/lee-tech/authentication/internal/models"
	authService "github.com/lee-tech/authentication/internal/service"
	coreConfig "github.com/lee-tech/core/config"
	coreLog "github.com/lee-tech/core/log"
	coreMiddleware "github.com/lee-tech/core/middleware"
	coreServer "github.com/lee-tech/core/server"
	"go.uber.org/zap"

	_ "github.com/lee-tech/authentication/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	var (
		additionalMiddleware      []mux.MiddlewareFunc
		adminAuthorizationBuilder = handlers.NewAdminAuthorizationBuilder()
	)

	checker, authorizationEnabled, err := coreMiddleware.NewAuthorizationCheckerFromConfig(cfg.Config, nil)
	if err != nil {
		log.Printf("failed to initialise authorization client: %v", err)
	} else if authorizationEnabled && checker != nil {
		additionalMiddleware = append(additionalMiddleware, coreMiddleware.WithAuthorizationChecker(checker))
	} else if cfg.DisableAuthorization {
		log.Println("authorization middleware disabled via DISABLE_AUTHORIZATION flag")
	}

	initialComponents := map[string]any{
		constants.ComponentKey.AuthenticationConfig:      cfg,
		constants.ComponentKey.AuthorizationEnabled:      authorizationEnabled,
		constants.ComponentKey.AdminAuthorizationBuilder: adminAuthorizationBuilder,
	}

	appOptions := &coreServer.HTTPAppOptions{
		Migrations:        []any{&models.User{}},
		InitialComponents: initialComponents,
	}
	if len(additionalMiddleware) > 0 {
		appOptions.AdditionalMiddleware = additionalMiddleware
	}

	app, err := coreServer.InitializeHTTPApp(cfg.Config, appOptions)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	cfg.RegisterOnConfigChange(func(newCfg *coreConfig.Config) {
		coreLog.Init(newCfg.LogLevel, newCfg.ServiceName, newCfg.ServiceVersion)
		app.Logger.Info("Configuration reloaded")
	})

	if watcher, err := config.NewWatcher(cfg.Config); err != nil {
		app.Logger.Warn("Failed to create config watcher", zap.Error(err))
	} else {
		watcher.Watch()
	}

	serviceComponent, ok := app.GetComponent(constants.ComponentKey.AuthenticationService)
	if !ok {
		log.Fatalf("component %s not found", constants.ComponentKey.AuthenticationService)
	}

	authSvc, ok := serviceComponent.(*authService.AuthenticationService)
	if !ok {
		log.Fatalf("component %s has unexpected type %T", constants.ComponentKey.AuthenticationService, serviceComponent)
	}

	if _, _, err := authSvc.BootstrapDefaultAdmin(); err != nil {
		log.Fatalf("failed to bootstrap default administrator: %v", err)
	}

	handler := handlers.NewAuthenticationHandler(authSvc, authorizationEnabled, adminAuthorizationBuilder)
	handler.RegisterRoutes(app.Router)

	app.Run()
}
