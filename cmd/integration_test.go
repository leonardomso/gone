//go:build integration

package cmd_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	buildBinaryOnce sync.Once
	builtBinaryPath string
	buildBinaryErr  error
)

func TestCheck_UsesConfigStructuredOutput_WhenNoLinksFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gonerc.yaml"), []byte("output:\n  format: json\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# No links here\n"), 0o644))

	result := runGone(t, tmpDir, "check", ".")
	require.Equal(t, 0, result.exitCode, result.stderr)
	require.Empty(t, strings.TrimSpace(result.stderr))

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.stdout), &payload))
	assert.Equal(t, float64(1), payload["total_files"])
	assert.Equal(t, float64(0), payload["total_links"])
}

func TestCheck_UsesConfigStructuredOutput_WhenAllLinksIgnored(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := `
output:
  format: json
ignore:
  domains: [ignored.example]
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gonerc.yaml"), []byte(config), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "README.md"),
		[]byte("[ignored](https://ignored.example/path)\n"),
		0o644,
	))

	result := runGone(t, tmpDir, "check", ".", "--show-ignored")
	require.Equal(t, 0, result.exitCode, result.stderr)
	require.Empty(t, strings.TrimSpace(result.stderr))

	var payload struct {
		TotalLinks int `json:"total_links"`
		Ignored    []struct {
			URL string `json:"url"`
		} `json:"ignored"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.stdout), &payload))
	assert.Equal(t, 1, payload.TotalLinks)
	require.Len(t, payload.Ignored, 1)
	assert.Equal(t, "https://ignored.example/path", payload.Ignored[0].URL)
}

func TestFix_Yes_UpdatesRedirectsAcrossFileTypes(t *testing.T) {
	t.Parallel()

	finalURL := ""
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/old":
			http.Redirect(w, r, finalURL, http.StatusMovedPermanently)
		case "/new":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer redirectServer.Close()

	finalURL = redirectServer.URL + "/new"
	oldURL := redirectServer.URL + "/old"

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "README.md"),
		[]byte("[docs]("+oldURL+")\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "links.json"),
		[]byte(`{"primary":"`+oldURL+`"}`),
		0o644,
	))

	result := runGone(t, tmpDir, "fix", ".", "--yes", "--types=md,json", "--no-config")
	require.Equal(t, 0, result.exitCode, result.stderr)
	assert.Contains(t, result.stdout, "Found 2 file(s) of type(s): md, json")
	assert.Contains(t, result.stdout, "Fixed 2 redirect(s) across 2 file(s):")

	mdContent, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	assert.NotContains(t, string(mdContent), oldURL)
	assert.Contains(t, string(mdContent), finalURL)

	jsonContent, err := os.ReadFile(filepath.Join(tmpDir, "links.json"))
	require.NoError(t, err)
	assert.NotContains(t, string(jsonContent), oldURL)
	assert.Contains(t, string(jsonContent), finalURL)
}

type cliResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runGone(t *testing.T, dir string, args ...string) cliResult {
	t.Helper()

	binaryPath := buildGoneBinary(t)
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()

	result := cliResult{
		stdout:   string(output),
		exitCode: 0,
	}

	if err == nil {
		return result
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.exitCode = exitErr.ExitCode()
		result.stderr = string(output)
		return result
	}

	t.Fatalf("run gone: %v\noutput:\n%s", err, string(output))
	return cliResult{}
}

func buildGoneBinary(t *testing.T) string {
	t.Helper()

	buildBinaryOnce.Do(func() {
		rootDir := repoRoot(t)
		tmpDir, err := os.MkdirTemp("", "gone-bin-*")
		if err != nil {
			buildBinaryErr = err
			return
		}

		builtBinaryPath = filepath.Join(tmpDir, "gone")
		if runtime.GOOS == "windows" {
			builtBinaryPath += ".exe"
		}

		cmd := exec.Command("go", "build", "-o", builtBinaryPath, ".")
		cmd.Dir = rootDir
		buildOutput, err := cmd.CombinedOutput()
		if err != nil {
			buildBinaryErr = &buildError{err: err, output: string(buildOutput)}
		}
	})

	require.NoError(t, buildBinaryErr)
	return builtBinaryPath
}

type buildError struct {
	err    error
	output string
}

func (e *buildError) Error() string {
	return e.err.Error() + "\n" + e.output
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Dir(filepath.Dir(file))
}
