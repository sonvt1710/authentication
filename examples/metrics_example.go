package examples

import (
	"log"

	redis "github.com/redis/go-redis/v9"
	coreServer "github.com/lee-tech/core/server"
	coreMiddleware "github.com/lee-tech/core/middleware"
)

// ExampleAuthenticationServiceWithMetrics demonstrates how to set up
// a service with authorization metrics, caching, and metrics endpoint.
//
// This is the recommended approach for all services.
func ExampleAuthenticationServiceWithMetrics() {
	// Step 1: Load your config (implementation depends on your config package)
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Step 2: Initialize app (metrics are automatically created!)
	app, err := coreServer.InitializeHTTPApp(cfg, &coreServer.HTTPAppOptions{})
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Step 3: Get the auto-created metrics instance
	metrics := app.GetAuthorizationMetrics()
	log.Printf("Metrics initialized for service: %s", cfg.ServiceName)

	// Step 4: Initialize Redis for caching (optional but recommended)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Step 5: Create authorization checker with cache
	checker, enabled, err := coreMiddleware.NewAuthorizationCheckerFromConfig(
		cfg,
		nil,         // Use default HTTP client
		redisClient, // Enables caching automatically
	)
	if err != nil {
		log.Printf("Failed to initialize authorization checker: %v", err)
	} else if enabled {
		log.Println("Authorization enabled with caching")
	}

	// Step 6: Wrap checker with service-wide metrics
	if checker != nil && metrics != nil {
		serviceAuth := coreMiddleware.NewServiceWideAuthorization(metrics)
		wrappedChecker := serviceAuth.WrapChecker(checker)

		// Register as middleware
		app.Router.Use(coreMiddleware.WithAuthorizationChecker(wrappedChecker))
		log.Println("Authorization middleware registered with metrics")
	}

	// Step 7: Register metrics endpoint
	app.RegisterMetricsEndpoint()
	log.Println("Metrics endpoint registered at /metrics/authorization")

	// Step 8: Register your service routes here
	// Example:
	// app.Router.HandleFunc("/api/v1/users", handleUsers).Methods(http.MethodGet)

	// Step 9: Start the server
	log.Println("Starting server with metrics and caching...")
	app.Run()
}

// loadConfig is a placeholder - implement based on your config package
func loadConfig() (*coreServer.HTTPApp, error) {
	// This should return your actual config
	// For example:
	// return config.Load()
	return nil, nil
}

// Alternative: Using the simpler wrapper approach
func ExampleSimpleMetricsSetup() {
	cfg, _ := loadConfig()

	app, _ := coreServer.InitializeHTTPApp(cfg, nil)

	// Get metrics
	metrics := app.GetAuthorizationMetrics()

	// Setup auth with cache
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	checker, _, _ := coreMiddleware.NewAuthorizationCheckerFromConfig(
		cfg,
		nil,
		redisClient,
	)

	// Wrap with metrics (simple one-liner)
	if checker != nil && metrics != nil {
		wrapped := coreMiddleware.NewServiceWideAuthorization(metrics).WrapChecker(checker)
		app.Router.Use(coreMiddleware.WithAuthorizationChecker(wrapped))
	}

	// Register metrics endpoint
	app.RegisterMetricsEndpoint()

	// Start
	app.Run()
}

// Manual metrics example (if you need custom metrics)
func ExampleManualMetricsAccess() {
	cfg, _ := loadConfig()
	app, _ := coreServer.InitializeHTTPApp(cfg, nil)

	// Access metrics
	metrics := app.GetAuthorizationMetrics()

	// Record custom events
	metrics.RecordRequest()
	metrics.RecordCacheHit()
	metrics.RecordDecision(true)
	metrics.RecordLatency(10 * time.Millisecond)

	// Get current metrics
	data := metrics.GetMetrics()
	log.Printf("Current metrics: %+v", data)

	// Reset metrics (e.g., for testing)
	// metrics.Reset()

	app.Run()
}
