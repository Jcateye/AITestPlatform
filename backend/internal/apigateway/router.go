package apigateway

import (
	"unified-ai-eval-platform/backend/internal/auth" // Adjust import path as necessary
	"unified-ai-eval-platform/backend/internal/configmanagement"

	"github.com/gin-gonic/gin"
)

// SetupRouter initializes the main Gin router for the API gateway.
// It includes public routes and authenticated routes.
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Public routes (e.g., login)
	authRoutes := router.Group("/auth")
	{
		// The LoadAdminCredentials function should be called at application startup,
		// for example, in the main.go file, before the router is set up.
		// auth.LoadAdminCredentials() // Call this in main.go

		authRoutes.POST("/login", auth.LoginHandler)
		// For MVP, logout might just clear a cookie, could be in authenticated group if it needs auth to clear server-side session
		authRoutes.POST("/logout", auth.LogoutHandler) // Or place under AdminRoutes if it needs auth
	}

	// Authenticated routes
	// All routes in this group will use the AuthMiddleware.
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(auth.AuthMiddleware()) // Apply the auth middleware
	{
		// Example of a protected route
		adminRoutes.GET("/dashboard", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Welcome to the admin dashboard!"})
		})

		// Other admin-only routes would go here:
		// adminRoutes.POST("/vendor-configs", ...)
		// adminRoutes.GET("/evaluation-jobs", ...)

		// Vendor Configuration Management Routes
		vendorRoutes := adminRoutes.Group("/vendors")
		{
			vendorRoutes.POST("", configmanagement.CreateVendorConfigHandler)
			vendorRoutes.GET("", configmanagement.ListVendorConfigsHandler)
			vendorRoutes.GET("/:id", configmanagement.GetVendorConfigHandler)
			vendorRoutes.PUT("/:id", configmanagement.UpdateVendorConfigHandler)
			vendorRoutes.DELETE("/:id", configmanagement.DeleteVendorConfigHandler)
		}

		// ASR Test Case Management Routes
		asrTestCaseRoutes := adminRoutes.Group("/asr-test-cases")
		{
			asrTestCaseRoutes.POST("", configmanagement.CreateASRTestCaseHandler)
			asrTestCaseRoutes.GET("", configmanagement.ListASRTestCasesHandler)
			asrTestCaseRoutes.GET("/:id", configmanagement.GetASRTestCaseHandler)
			asrTestCaseRoutes.PUT("/:id", configmanagement.UpdateASRTestCaseHandler)
			asrTestCaseRoutes.DELETE("/:id", configmanagement.DeleteASRTestCaseHandler)
		}

		// Evaluation Job Management Routes
		jobRoutes := adminRoutes.Group("/jobs")
		{
			jobRoutes.POST("/asr", jobmanagement.CreateASRJobHandler) // Specific for ASR jobs
			jobRoutes.GET("", jobmanagement.ListJobsHandler)
			jobRoutes.GET("/:id", jobmanagement.GetJobHandler)
			jobRoutes.GET("/:id/results", jobmanagement.GetJobResultsHandler)
		}
	}

	return router
}

// Note: The actual server startup (e.g., in backend/cmd/server/main.go) would look something like:
/*
package main

import (
	"log"
	"os" // For getting ENV variable for DB connection
	"unified-ai-eval-platform/backend/internal/apigateway"
	"unified-ai-eval-platform/backend/internal/auth"
	"unified-ai-eval-platform/backend/internal/configmanagement"
	"unified-ai-eval-platform/backend/internal/datastore"
	"unified-ai-eval-platform/backend/internal/jobmanagement"  // Added for Job routes
	"unified-ai-eval-platform/backend/internal/objectstore"
)

func main() {
	// Load configurations at startup
	auth.LoadAdminCredentials() // Crucial: Load admin credentials

	// Initialize DB connection
	// In a real app, use a proper config management solution (e.g., Viper)
	// For now, using an environment variable for the DSN
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE") // e.g., "disable" for local dev

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres" // Default user
	}
	if dbPassword == "" {
		log.Println("WARNING: DB_PASSWORD environment variable not set.")
		// dbPassword might be intentionally empty for some local setups, but usually not for prod.
	}
	if dbName == "" {
		dbName = "ai_eval_platform_db" // Example DB name
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}
	
	// dataSourceName := "host=localhost user=youruser password=yourpassword dbname=yourdbname sslmode=disable"
	dataSourceName := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)


	if err := datastore.InitDB(dataSourceName); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer datastore.DB.Close()

	// Pass the DB instance to the handlers if needed (using InitHandlers as an example)
	// This step is somewhat redundant given the current global DB in datastore, but good for showing intent.
	configmanagement.InitHandlers(datastore.DB) // For vendor_handlers

	// Initialize MinIO Client
	if err := objectstore.InitMinioClient(); err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}


	// Setup router
	router := apigateway.SetupRouter()

	// Start server
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080" // Default port
	}
	log.Printf("Starting server on :%s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
*/
