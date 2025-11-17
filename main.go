package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type ContainerState struct {
	Status string `json:"Status"`
	Health *struct {
		Status string `json:"Status"`
	} `json:"Health"`
}

func getContainerState(containerName string) (string, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{json .State}}", containerName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	var state ContainerState
	if err := json.Unmarshal(output, &state); err != nil {
		return "", fmt.Errorf("failed to parse container state: %w", err)
	}

	if state.Status != "running" {
		return "exited", nil
	}

	if state.Health == nil {
		return "running", nil
	}

	if state.Health.Status == "healthy" {
		return "healthy", nil
	}

	return "unhealthy", nil
}

func getAllContainerStates() (map[string]string, error) {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	containerNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	states := make(map[string]string)

	for _, containerName := range containerNames {
		containerName = strings.TrimSpace(containerName)
		if containerName == "" {
			continue
		}

		if len(containerName) > 0 && containerName[0] == '/' {
			containerName = containerName[1:]
		}

		state, err := getContainerState(containerName)
		if err != nil {
			continue
		}

		states[containerName] = state
	}

	return states, nil
}

func main() {
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(middleware.RemoveTrailingSlash())

	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 10 * time.Second,
	}))

	limiterStore := middleware.NewRateLimiterMemoryStore(2)
	e.Use(middleware.RateLimiter(limiterStore))

	e.GET("/metrics", func(c echo.Context) error {
		states, err := getAllContainerStates()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, states)
	})

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "9066"
	}

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	fmt.Println("Server starting on", addr)
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
