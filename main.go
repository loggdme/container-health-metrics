package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const dockerSocketPath = "/var/run/docker.sock"

var dockerClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", dockerSocketPath)
		},
	},
	Timeout: 5 * time.Second,
}

type ContainerState struct {
	Status string `json:"Status"`
	Health *struct {
		Status string `json:"Status"`
	} `json:"Health"`
}

type ContainerInfo struct {
	ID    string         `json:"Id"`
	Names []string       `json:"Names"`
	State ContainerState `json:"State"`
}

type ContainerListItem struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
}

func getContainerState(containerID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost/containers/%s/json", containerID), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := dockerClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to inspect container: status %d", resp.StatusCode)
	}

	var containerInfo ContainerInfo
	if err := json.NewDecoder(resp.Body).Decode(&containerInfo); err != nil {
		return "", fmt.Errorf("failed to parse container info: %w", err)
	}

	state := containerInfo.State
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
	req, err := http.NewRequest("GET", "http://localhost/containers/json?all=true", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := dockerClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list containers: status %d", resp.StatusCode)
	}

	var containers []ContainerListItem
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("failed to parse containers list: %w", err)
	}

	states := make(map[string]string)

	for _, container := range containers {
		if len(container.Names) == 0 {
			continue
		}

		containerName := strings.TrimPrefix(container.Names[0], "/")
		if containerName == "" {
			continue
		}

		state, err := getContainerState(container.ID)
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
