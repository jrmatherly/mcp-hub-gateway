package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/catalog"
)

type remoteMCPClient struct {
	config      *catalog.ServerConfig
	client      *mcp.Client
	session     *mcp.ClientSession
	roots       []*mcp.Root
	initialized atomic.Bool
}

func NewRemoteMCPClient(config *catalog.ServerConfig) Client {
	return &remoteMCPClient{
		config: config,
	}
}

func (c *remoteMCPClient) Initialize(
	ctx context.Context,
	_ *mcp.InitializeParams,
	_ bool,
	_ *mcp.ServerSession,
	_ *mcp.Server,
	_ CapabilityRefresher,
) error {
	if c.initialized.Load() {
		return fmt.Errorf("client already initialized")
	}

	// Read configuration.
	var (
		url       string
		transport string
	)
	if c.config.Spec.SSEEndpoint != "" {
		// Deprecated
		url = c.config.Spec.SSEEndpoint
		transport = "sse"
	} else {
		url = c.config.Spec.Remote.URL
		transport = c.config.Spec.Remote.Transport
	}

	// Secrets to env
	env := map[string]string{}
	for _, secret := range c.config.Spec.Secrets {
		env[secret.Env] = c.config.Secrets[secret.Name]
	}

	// Headers
	headers := map[string]string{}
	for k, v := range c.config.Spec.Remote.Headers {
		headers[k] = expandEnv(v, env)
	}

	var mcpTransport mcp.Transport
	var err error

	// Create HTTP client with custom headers
	httpClient := &http.Client{
		Transport: &headerRoundTripper{
			base:    http.DefaultTransport,
			headers: headers,
		},
	}

	switch strings.ToLower(transport) {
	case "sse":
		mcpTransport = &mcp.SSEClientTransport{
			Endpoint:   url,
			HTTPClient: httpClient,
		}
	case "http", "streamable", "streaming", "streamable-http":
		mcpTransport = &mcp.StreamableClientTransport{
			Endpoint:   url,
			HTTPClient: httpClient,
		}
	default:
		return fmt.Errorf("unsupported remote transport: %s", transport)
	}

	c.client = mcp.NewClient(&mcp.Implementation{
		Name:    "docker-mcp-gateway",
		Version: "1.0.0",
	}, nil)

	c.client.AddRoots(c.roots...)

	session, err := c.client.Connect(ctx, mcpTransport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.session = session
	c.initialized.Store(true)

	return nil
}

func (c *remoteMCPClient) Session() *mcp.ClientSession { return c.session }
func (c *remoteMCPClient) GetClient() *mcp.Client      { return c.client }

func (c *remoteMCPClient) AddRoots(roots []*mcp.Root) {
	if c.initialized.Load() {
		c.client.AddRoots(roots...)
	}
	c.roots = roots
}

func expandEnv(value string, secrets map[string]string) string {
	return os.Expand(value, func(name string) string {
		return secrets[name]
	})
}

// headerRoundTripper is an http.RoundTripper that adds custom headers to all requests
type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	newReq := req.Clone(req.Context())
	// Add custom headers
	for key, value := range h.headers {
		newReq.Header.Set(key, value)
	}
	return h.base.RoundTrip(newReq)
}
