package api

import (
	"time"

	"github.com/CristiGvl/picoHWMon/internal/cpu"
	"github.com/CristiGvl/picoHWMon/internal/disk"
	"github.com/CristiGvl/picoHWMon/internal/fan"
	"github.com/CristiGvl/picoHWMon/internal/gpu"
	"github.com/CristiGvl/picoHWMon/internal/memory"
	"github.com/CristiGvl/picoHWMon/internal/overclock"
	"github.com/CristiGvl/picoHWMon/internal/platform"
	"github.com/CristiGvl/picoHWMon/internal/temps"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// Server represents the API server
type Server struct {
	app                 *fiber.App
	cpuReader           cpu.Reader
	gpuReader           gpu.Reader
	memoryReader        memory.Reader
	diskReader          disk.Reader
	tempsReader         temps.Reader
	fanController       fan.Controller
	overclockController overclock.Controller
}

// NewServer creates a new API server
func NewServer() (*Server, error) {
	// Validate platform support
	if err := platform.ValidateSupport(); err != nil {
		return nil, err
	}

	app := fiber.New(fiber.Config{
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		IdleTimeout:        120 * time.Second,
		DisableKeepalive:   false,
		EnableIPValidation: false,
		ServerHeader:       "picoHWMon",
		AppName:            "picoHardwareMonitor v1.0",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "*",
		AllowCredentials: false,
		ExposeHeaders:    "Content-Length,Content-Type,Access-Control-Allow-Origin",
		MaxAge:           86400, // 24 hours
	}))

	// Add explicit CORS headers middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
		c.Set("Access-Control-Allow-Headers", "*")

		// Handle preflight requests
		if c.Method() == "OPTIONS" {
			return c.SendStatus(204)
		}

		return c.Next()
	})

	server := &Server{
		app:                 app,
		cpuReader:           cpu.NewReader(),
		gpuReader:           gpu.NewReader(),
		memoryReader:        memory.NewReader(),
		diskReader:          disk.NewReader(),
		tempsReader:         temps.NewReader(),
		fanController:       fan.NewController(),
		overclockController: overclock.NewController(),
	}

	server.setupRoutes()
	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	api := s.app.Group("/api")

	// System information endpoints
	api.Get("/cpu", s.getCPU)
	api.Get("/gpu", s.getGPU)
	api.Get("/memory", s.getMemory)
	api.Get("/disk", s.getDisk)
	api.Get("/temps", s.getTemps)

	// Fan control endpoints
	api.Get("/fan", s.getFans)
	api.Get("/fan/:id/settings", s.getFanSettings)
	api.Post("/fan/:id/settings", s.setFanSettings)

	// GPU overclocking endpoints
	api.Get("/gpu/:id/overclock", s.getGPUOverclock)
	api.Post("/gpu/:id/overclock", s.setGPUOverclock)

	// Overclocking endpoints
	api.Get("/overclock/profiles", s.getOverclockProfiles)
	api.Post("/overclock/profiles", s.saveOverclockProfile)
	api.Post("/overclock/profiles/:name/load", s.loadOverclockProfile)
	api.Get("/overclock/:deviceId", s.getOverclockSettings)
	api.Post("/overclock", s.setOverclockSettings)

	// Health check
	api.Get("/health", s.healthCheck)
}

// Start starts the API server
func (s *Server) Start(address string) error {
	return s.app.Listen(address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

// Health check endpoint
func (s *Server) healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"platform":  platform.GetOS(),
		"timestamp": time.Now().Unix(),
	})
}
