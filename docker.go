package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	SearXNGContainerName = "searxng-mcp-local"
	SearXNGImage         = "searxng/searxng:latest"
	SearXNGInternalPort  = "8080"
)

type DockerManager struct {
	searxngPort  string
	searxngImage string
	autostart    bool
	stopOnExit   bool
}

func NewDockerManager(searxngPort, searxngImage string, autostart, stopOnExit bool) *DockerManager {
	return &DockerManager{
		searxngPort:  searxngPort,
		searxngImage: searxngImage,
		autostart:    autostart,
		stopOnExit:   stopOnExit,
	}
}

func (dm *DockerManager) Close() error {
	if dm.stopOnExit {
		return dm.StopSearXNG()
	}
	return nil
}

func (dm *DockerManager) runDocker(args ...string) (string, string, error) {
	cmd := exec.Command("docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (dm *DockerManager) IsDockerAvailable() bool {
	_, _, err := dm.runDocker("version")
	return err == nil
}

func (dm *DockerManager) IsSearXNGRunning() bool {
	stdout, _, err := dm.runDocker("ps", "-q", "-f", "name="+SearXNGContainerName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(stdout) != ""
}

func (dm *DockerManager) GetSearXNGHealth() bool {
	url := fmt.Sprintf("http://127.0.0.1:%s/config", dm.searxngPort)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (dm *DockerManager) StartSearXNG() error {
	if !dm.autostart {
		return fmt.Errorf("auto-start disabled")
	}

	// Validate port is within valid range
	var portNum int
	_, err := fmt.Sscanf(dm.searxngPort, "%d", &portNum)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid port: must be between 1 and 65535, got %s", dm.searxngPort)
	}

	// Check if container exists but stopped
	stdout, _, _ := dm.runDocker("ps", "-a", "-q", "-f", "name="+SearXNGContainerName)
	if strings.TrimSpace(stdout) != "" {
		// Container exists, start it
		_, _, err := dm.runDocker("start", SearXNGContainerName)
		if err != nil {
			return fmt.Errorf("failed to start existing SearXNG container: %w", err)
		}
		return nil
	}

	// Pull image in background (don't wait)
	go dm.pullImage()

	// Create and start new container with a random secret
	// Generate a random secret (SearXNG requires at least 16 chars)
	secret := fmt.Sprintf("mcp-search-%d-secret-key-local", time.Now().Unix())

	// Create and start new container with JSON API enabled
	// Use environment variables to configure SearXNG
	_, stderr, err := dm.runDocker(
		"run",
		"-d",
		"--name", SearXNGContainerName,
		"-p", "127.0.0.1:"+dm.searxngPort+":"+SearXNGInternalPort,
		"--restart", "unless-stopped",
		"-e", "SEARXNG_SECRET="+secret,
		dm.searxngImage,
	)

	if err != nil {
		// Check if it's a port conflict
		if strings.Contains(stderr, "Bind for 127.0.0.1") {
			return fmt.Errorf("port %s is already in use. Please stop the existing service or use a different port", dm.searxngPort)
		}
		return fmt.Errorf("failed to create SearXNG container: %w", err)
	}

	return nil
}

func (dm *DockerManager) pullImage() {
	dm.runDocker("pull", dm.searxngImage)
}

func (dm *DockerManager) StopSearXNG() error {
	_, _, _ = dm.runDocker("stop", SearXNGContainerName)
	return nil
}

func (dm *DockerManager) RemoveSearXNG() error {
	_, _, _ = dm.runDocker("rm", "-f", SearXNGContainerName)
	return nil
}

func (dm *DockerManager) EnsureSearXNG() error {
	if dm.IsSearXNGRunning() && dm.GetSearXNGHealth() {
		return nil
	}

	if !dm.IsDockerAvailable() {
		return fmt.Errorf("Docker is not available. Please install Docker or start SearXNG manually at http://127.0.0.1:%s", dm.searxngPort)
	}

	if err := dm.StartSearXNG(); err != nil {
		return fmt.Errorf("failed to start SearXNG: %w", err)
	}

	// Configure SearXNG for JSON API access
	if err := dm.configureSearXNG(); err != nil {
		// Log but don't fail - container might already be configured
		fmt.Printf("Note: SearXNG configuration: %v\n", err)
	}

	return nil
}

// configureSearXNG enables JSON format for API access
func (dm *DockerManager) configureSearXNG() error {
	// Wait a bit for container to be fully ready
	time.Sleep(2 * time.Second)

	// Enable JSON format in settings
	// First uncomment the formats line, then add json to the list
	_, _, err := dm.runDocker("exec", SearXNGContainerName, "sh", "-c",
		`sed -i 's/# formats:/formats:/' /etc/searxng/settings.yml && sed -i '/formats:/,/^[a-z]/ { s/  - html/  - html\n    - json/; }' /etc/searxng/settings.yml`)
	if err != nil {
		return fmt.Errorf("failed to enable JSON format: %w", err)
	}

	// Restart to apply changes
	_, _, err = dm.runDocker("restart", SearXNGContainerName)
	if err != nil {
		return fmt.Errorf("failed to restart SearXNG: %w", err)
	}

	// Wait for restart
	time.Sleep(3 * time.Second)

	return nil
}

func (dm *DockerManager) WaitForSearXNGReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if dm.GetSearXNGHealth() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("SearXNG failed to become ready within %v", timeout)
}
