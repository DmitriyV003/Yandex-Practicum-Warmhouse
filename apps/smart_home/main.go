package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smarthome/db"
	_ "smarthome/docs"
	"smarthome/handlers"
	"smarthome/services"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Smart Home API
// @version         1.0
// @description     API для управления датчиками умного дома
// @host            localhost:8080
// @BasePath        /api/v1
func main() {
	// Set up database connection
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smarthome")
	database, err := db.New(dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer database.Close()

	log.Println("Connected to database successfully")

	// Initialize temperature service
	temperatureAPIURL := getEnv("TEMPERATURE_API_URL", "http://temperature-api:8081")
	temperatureService := services.NewTemperatureService(temperatureAPIURL)
	log.Printf("Temperature service initialized with API URL: %s\n", temperatureAPIURL)

	// Initialize gRPC clients for microservices
	var deviceClient *services.DeviceClient
	deviceServiceURL := getEnv("DEVICE_SERVICE_URL", "device-service:8082")
	deviceClient, err = services.NewDeviceClient(deviceServiceURL)
	if err != nil {
		log.Printf("Warning: device-service unavailable: %v", err)
	} else {
		defer deviceClient.Close()
		log.Printf("Connected to device-service at %s", deviceServiceURL)
	}

	var telemetryClient *services.TelemetryClient
	telemetryServiceURL := getEnv("TELEMETRY_SERVICE_URL", "telemetry-service:8083")
	telemetryClient, err = services.NewTelemetryClient(telemetryServiceURL)
	if err != nil {
		log.Printf("Warning: telemetry-service unavailable: %v", err)
	} else {
		defer telemetryClient.Close()
		log.Printf("Connected to telemetry-service at %s", telemetryServiceURL)
	}

	// Initialize router
	router := gin.Default()

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API routes
	apiRoutes := router.Group("/api/v1")

	// Register sensor routes
	sensorHandler := handlers.NewSensorHandler(database, temperatureService, deviceClient, telemetryClient)
	sensorHandler.RegisterRoutes(apiRoutes)

	// Start server
	srv := &http.Server{
		Addr:    getEnv("PORT", ":8080"),
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Server starting on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	log.Println("Server exited properly")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
