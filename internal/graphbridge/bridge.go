package graphbridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Bridge communicates with the teraflow-graphrag Python module via subprocess.
type Bridge struct {
	projectRoot string
}

// New creates a new Bridge for the given project root.
func New(projectRoot string) *Bridge {
	return &Bridge{projectRoot: projectRoot}
}

// Available checks if the teraflow-graphrag Python module is available.
// Detection order (design doc ch.12):
//   - {project_root}/graphrag/pyproject.toml exists
//   - python3 -m teraflow_graphrag --version succeeds
func (b *Bridge) Available() bool {
	pyprojectPath := filepath.Join(b.projectRoot, "graphrag", "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		return true
	}

	cmd := exec.Command("python3", "-m", "teraflow_graphrag", "--version")
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

// Execute sends a command to the Python subprocess and returns the response.
func (b *Bridge) Execute(req Request) (*Response, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("graphbridge: marshal request: %w", err)
	}

	cmd := exec.Command("python3", "-m", "teraflow_graphrag")
	cmd.Dir = b.projectRoot
	cmd.Stdin = bytes.NewReader(payload)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("graphbridge: subprocess failed: %w\nstderr: %s", err, stderr.String())
	}

	var resp Response
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("graphbridge: unmarshal response: %w\nstdout: %s", err, stdout.String())
	}
	if !resp.OK {
		return nil, fmt.Errorf("graphbridge: python error: %s", resp.Error)
	}
	return &resp, nil
}
