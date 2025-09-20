package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/catalog"
)

func thisIsAnIntegrationTest(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test.")
	}
}

// runDockerMCPWithStreams separates stdout and stderr for proper protocol testing
func runDockerMCPWithStreams(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	args = append([]string{"mcp"}, args...)
	t.Logf("[%s]", strings.Join(args, " "))

	cmd := exec.CommandContext(t.Context(), "docker", args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		t.Logf("Command failed: %v", err)
		t.Logf("Stdout: %s", stdoutBuf.String())
		t.Logf("Stderr: %s", stderrBuf.String())
		require.NoError(t, err)
	}

	return stdoutBuf.String(), stderrBuf.String()
}

// runDockerMCP maintains backward compatibility by combining stdout and stderr
func runDockerMCP(t *testing.T, args ...string) string {
	t.Helper()
	stdout, stderr := runDockerMCPWithStreams(t, args...)
	// For backward compatibility, combine output (stderr first, then stdout)
	// This matches the behavior of CombinedOutput which interleaves streams
	return stderr + stdout
}

func writeFile(t *testing.T, parent, name string, content string) {
	t.Helper()
	path := filepath.Join(parent, name)
	require.NoError(t, os.MkdirAll(filepath.Base(parent), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// createClickhouseCatalogFile creates a temporary catalog file containing only the clickhouse server entry
func createClickhouseCatalogFile(t *testing.T, tempDir string) string {
	t.Helper()

	// Create a minimal catalog using raw YAML content
	catalogYAML := `registry:
  clickhouse:
    description: Official ClickHouse MCP Server.
    title: Official ClickHouse MCP Server
    type: server
    dateAdded: "2025-06-12T18:00:16Z"
    image: mcp/clickhouse@sha256:3a18fb4687c2f08364fd27be4bb3a7f33e2c77b22d3bca2760d22dcb73e47108
    ref: ""
    readme: http://desktop.docker.com/mcp/catalog/v2/readme/clickhouse.md
    toolsUrl: http://desktop.docker.com/mcp/catalog/v2/tools/clickhouse.json
    source: https://github.com/ClickHouse/mcp-clickhouse/tree/main
    upstream: https://github.com/ClickHouse/mcp-clickhouse
    icon: https://avatars.githubusercontent.com/u/54801242?v=4
    tools:
      - name: list_databases
      - name: list_tables
      - name: run_select_query
    secrets:
      - name: clickhouse.password
        env: CLICKHOUSE_PASSWORD
        example: <YOUR_CLICKHOUSE_PASSWORD_HERE>
    env:
      - name: CLICKHOUSE_HOST
        value: '{{clickhouse.host}}'
      - name: CLICKHOUSE_PORT
        value: '{{clickhouse.port}}'
      - name: CLICKHOUSE_USER
        value: '{{clickhouse.user}}'
      - name: CLICKHOUSE_SECURE
        value: '{{clickhouse.secure}}'
      - name: CLICKHOUSE_VERIFY
        value: '{{clickhouse.verify}}'
      - name: CLICKHOUSE_CONNECT_TIMEOUT
        value: '{{clickhouse.connect_timeout}}'
      - name: CLICKHOUSE_SEND_RECEIVE_TIMEOUT
        value: '{{clickhouse.send_receive_timeout}}'
    prompts: 0
    resources: {}
    config:
      - name: clickhouse
        description: Configure the connection to ClickHouse
        type: object
        properties:
          host:
            type: string
          port:
            type: string
          user:
            type: string
          secure:
            type: string
          verify:
            type: string
          connect_timeout:
            type: string
          send_receive_timeout:
            type: string
    metadata:
      pulls: 10413
      stars: 2
      githubStars: 519
      category: database
      tags:
        - database
        - clickhouse
      license: Apache License 2.0
      owner: ClickHouse
`

	// Write to temporary file
	catalogFile := filepath.Join(tempDir, "clickhouse-catalog.yaml")
	require.NoError(t, os.WriteFile(catalogFile, []byte(catalogYAML), 0o644))

	return catalogFile
}

func TestIntegrationVersion(t *testing.T) {
	thisIsAnIntegrationTest(t)
	out := runDockerMCP(t, "version")
	assert.NotEmpty(t, out)
}

func TestIntegrationCatalogLs(t *testing.T) {
	thisIsAnIntegrationTest(t)
	out := runDockerMCP(t, "catalog", "ls")
	assert.Contains(t, out, "docker-mcp: Docker MCP Catalog")
}

func TestIntegrationCatalogShow(t *testing.T) {
	thisIsAnIntegrationTest(t)
	out := runDockerMCP(t, "catalog", "show")
	assert.Contains(t, out, "playwright:")
}

func TestIntegrationDryRunEmpty(t *testing.T) {
	thisIsAnIntegrationTest(t)
	stdout, stderr := runDockerMCPWithStreams(t, "gateway", "run", "--dry-run", "--servers=")

	// Operational logs should be in stderr
	// Note: Even with --servers=, the gateway loads default servers
	assert.Contains(t, stderr, "- Those servers are enabled")
	assert.Contains(t, stderr, "Initialized in")

	// stdout should be empty for dry-run (no protocol messages)
	assert.Empty(t, stdout, "stdout should be empty in dry-run mode")
}

func TestIntegrationDryRunFetch(t *testing.T) {
	thisIsAnIntegrationTest(t)
	stdout, stderr := runDockerMCPWithStreams(
		t,
		"gateway",
		"run",
		"--dry-run",
		"--servers=fetch",
		"--catalog="+catalog.DockerCatalogURL,
	)

	// Check stderr for operational messages
	assert.Contains(t, stderr, "- Those servers are enabled: fetch")
	assert.Contains(t, stderr, "fetch: (1 tools)")
	assert.Contains(t, stderr, "Initialized in")

	// stdout should be empty for dry-run
	assert.Empty(t, stdout, "stdout should be empty in dry-run mode")
}

func TestIntegrationCallToolClickhouse(t *testing.T) {
	thisIsAnIntegrationTest(t)
	tmp := t.TempDir()
	writeFile(t, tmp, ".env", "clickhouse.password=")
	writeFile(
		t,
		tmp,
		"config.yaml",
		"clickhouse:\n  host: sql-clickhouse.clickhouse.com\n  user: demo\n",
	)

	// Create temporary catalog file with only the clickhouse entry
	catalogFile := createClickhouseCatalogFile(t, tmp)

	gatewayArgs := []string{
		"--servers=clickhouse",
		"--catalog=" + catalogFile,
		"--secrets=" + filepath.Join(tmp, ".env"),
		"--config=" + filepath.Join(tmp, "config.yaml"),
		"--verbose",
	}

	stdout, stderr := runDockerMCPWithStreams(
		t,
		"tools",
		"call",
		"--gateway-arg="+strings.Join(gatewayArgs, ","),
		"list_databases",
	)

	// Protocol response should be in stdout
	assert.Contains(t, stdout, "amazon")
	assert.Contains(t, stdout, "bluesky")
	assert.Contains(t, stdout, "country")

	// Operational logs can be in stderr (ignored for protocol tests)
	_ = stderr
}

func TestIntegrationCallToolDuckDuckDb(t *testing.T) {
	thisIsAnIntegrationTest(t)
	gatewayArgs := []string{
		"--servers=duckduckgo",
		"--catalog=" + catalog.DockerCatalogURL,
	}

	stdout, stderr := runDockerMCPWithStreams(
		t,
		"tools",
		"call",
		"--gateway-arg="+strings.Join(gatewayArgs, ","),
		"search",
		"query=Docker",
	)

	// Protocol response should be in stdout
	assert.Contains(t, stdout, "Found 10 search results")

	// Operational logs can be in stderr (ignored for protocol tests)
	_ = stderr
}
