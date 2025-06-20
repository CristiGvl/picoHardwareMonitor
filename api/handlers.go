package api

import (
	"context"
	"strconv"
	"time"

	"github.com/CristiGvl/picoHWMon/internal/fan"
	"github.com/CristiGvl/picoHWMon/internal/gpu"
	"github.com/CristiGvl/picoHWMon/internal/overclock"
	"github.com/gofiber/fiber/v2"
)

// CPU endpoint
func (s *Server) getCPU(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := s.cpuReader.GetInfo(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(info)
}

// GPU endpoint
func (s *Server) getGPU(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := s.gpuReader.GetInfo(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(info)
}

// Memory endpoint
func (s *Server) getMemory(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := s.memoryReader.GetInfo(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(info)
}

// Disk endpoint
func (s *Server) getDisk(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := s.diskReader.GetInfo(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(info)
}

// Temperature endpoint
func (s *Server) getTemps(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := s.tempsReader.GetInfo(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(info)
}

// Fan endpoints
func (s *Server) getFans(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fans, err := s.fanController.GetFans(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fans)
}

func (s *Server) setFanSettings(c *fiber.Ctx) error {
	fanID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid fan ID"})
	}

	var settings fan.Settings
	if err := c.BodyParser(&settings); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.fanController.SetSettings(ctx, fanID, &settings); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success"})
}

// Get fan settings endpoint
func (s *Server) getFanSettings(c *fiber.Ctx) error {
	fanID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid fan ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := s.fanController.GetSettings(ctx, fanID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(settings)
}

// GPU overclocking endpoints
func (s *Server) getGPUOverclock(c *fiber.Ctx) error {
	deviceID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid device ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := s.gpuReader.GetOverclockSettings(ctx, deviceID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(settings)
}

func (s *Server) setGPUOverclock(c *fiber.Ctx) error {
	deviceID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid device ID"})
	}

	var reqSettings struct {
		CoreClockOffset   int `json:"core_clock_offset_mhz"`
		MemoryClockOffset int `json:"memory_clock_offset_mhz"`
		PowerLimit        int `json:"power_limit_percent"`
		FanSpeed          int `json:"fan_speed_percent"`
	}

	if err := c.BodyParser(&reqSettings); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert to GPU overclock settings type
	settings := &gpu.OverclockSettings{
		CoreClockOffset:   reqSettings.CoreClockOffset,
		MemoryClockOffset: reqSettings.MemoryClockOffset,
		PowerLimit:        reqSettings.PowerLimit,
		FanSpeed:          reqSettings.FanSpeed,
	}

	result, err := s.gpuReader.SetOverclockSettings(ctx, deviceID, settings)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Return detailed result with status code based on success
	if result.Success {
		return c.JSON(result)
	} else {
		return c.Status(422).JSON(result) // Unprocessable Entity for partial failures
	}
}

// Overclocking endpoints
func (s *Server) getOverclockSettings(c *fiber.Ctx) error {
	deviceID, err := strconv.Atoi(c.Params("deviceId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid device ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := s.overclockController.GetSettings(ctx, deviceID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(settings)
}

func (s *Server) setOverclockSettings(c *fiber.Ctx) error {
	var settings overclock.Settings
	if err := c.BodyParser(&settings); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.overclockController.SetSettings(ctx, &settings); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success"})
}

func (s *Server) getOverclockProfiles(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	profiles, err := s.overclockController.GetProfiles(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(profiles)
}

func (s *Server) saveOverclockProfile(c *fiber.Ctx) error {
	var profile overclock.Profile
	if err := c.BodyParser(&profile); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.overclockController.SaveProfile(ctx, &profile); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success"})
}

func (s *Server) loadOverclockProfile(c *fiber.Ctx) error {
	profileName := c.Params("name")
	if profileName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "profile name required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.overclockController.LoadProfile(ctx, profileName); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "success"})
}
