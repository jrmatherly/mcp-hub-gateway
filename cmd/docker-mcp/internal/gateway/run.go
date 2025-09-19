package gateway

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/docker"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/health"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/interceptors"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/telemetry"
)

type ServerSessionCache struct {
	Roots []*mcp.Root
}

// type SubsAction int

// const (
// subscribe   SubsAction = 0
// unsubscribe SubsAction = 1
// )

// type SubsMessage struct {
// uri    string
// action SubsAction
// ss     *mcp.ServerSession
// }

type Gateway struct {
	Options
	docker        docker.Client
	configurator  Configurator
	configuration Configuration
	clientPool    *clientPool
	mcpServer     *mcp.Server
	health        health.State
	// subsChannel  chan SubsMessage

	sessionCacheMu sync.RWMutex
	sessionCache   map[*mcp.ServerSession]*ServerSessionCache

	// Track registered capabilities for cleanup during reload
	registeredToolNames            []string
	registeredPromptNames          []string
	registeredResourceURIs         []string
	registeredResourceTemplateURIs []string
}

func NewGateway(config Config, docker docker.Client) *Gateway {
	g := &Gateway{
		Options: config.Options,
		docker:  docker,
		configurator: &FileBasedConfiguration{
			ServerNames:        config.ServerNames,
			CatalogPath:        config.CatalogPath,
			RegistryPath:       config.RegistryPath,
			ConfigPath:         config.ConfigPath,
			SecretsPath:        config.SecretsPath,
			ToolsPath:          config.ToolsPath,
			OciRef:             config.OciRef,
			MCPRegistryServers: config.MCPRegistryServers,
			Watch:              config.Watch,
			Central:            config.Central,
			docker:             docker,
		},
		sessionCache: make(map[*mcp.ServerSession]*ServerSessionCache),
	}
	g.clientPool = newClientPool(config.Options, docker, g)
	return g
}

func (g *Gateway) Run(ctx context.Context) error {
	// Initialize telemetry
	telemetry.Init()

	// Record gateway start
	transportMode := "stdio"
	if g.Port != 0 {
		transportMode = "sse"
	}
	telemetry.RecordGatewayStart(ctx, transportMode)

	// Start periodic metric export for long-running gateway
	// This is critical because Docker CLI's ManualReader only exports on shutdown
	// which is inappropriate for gateways that can run for hours, days, or weeks
	// ALL gateway run commands are long-lived regardless of transport (stdio, sse, streaming)
	// Even stdio mode runs as long as the client (e.g., Claude Code) is connected
	if !g.DryRun {
		go g.periodicMetricExport(ctx)
	}

	defer g.clientPool.Close()
	defer func() {
		// Clean up all session cache entries
		g.sessionCacheMu.Lock()
		g.sessionCache = make(map[*mcp.ServerSession]*ServerSessionCache)
		g.sessionCacheMu.Unlock()
	}()

	start := time.Now()

	// Listen as early as possible to not lose client connections.
	var ln net.Listener
	if port := g.Port; port != 0 {
		var (
			lc  net.ListenConfig
			err error
		)
		ln, err = lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return err
		}
	}

	// Read the configuration.
	configuration, configurationUpdates, stopConfigWatcher, err := g.configurator.Read(ctx)
	g.configuration = configuration
	if err != nil {
		return err
	}
	defer func() { _ = stopConfigWatcher() }()

	// Parse interceptors
	var parsedInterceptors []interceptors.Interceptor
	if len(g.Interceptors) > 0 {
		var err error
		parsedInterceptors, err = interceptors.Parse(g.Interceptors)
		if err != nil {
			return fmt.Errorf("parsing interceptors: %w", err)
		}
		log("- Interceptors enabled:", strings.Join(g.Interceptors, ", "))
	}

	g.mcpServer = mcp.NewServer(&mcp.Implementation{
		Name:    "Docker AI MCP Gateway",
		Version: "2.0.1",
	}, &mcp.ServerOptions{
		SubscribeHandler: func(_ context.Context, req *mcp.SubscribeRequest) error {
			log("- Client subscribed to URI:", req.Params.URI)
			// The MCP SDK doesn't provide ServerSession in SubscribeHandler because it already
			// keeps track of the mapping between ServerSession and subscribed resources in the Server
			// g.subsChannel <- SubsMessage{uri: req.Params.URI, action: subscribe , ss: ss}
			return nil
		},
		UnsubscribeHandler: func(_ context.Context, req *mcp.UnsubscribeRequest) error {
			log("- Client unsubscribed from URI:", req.Params.URI)
			// The MCP SDK doesn't provide ServerSession in UnsubscribeHandler because it already
			// keeps track of the mapping ServerSession and subscribed resources in the Server
			// g.subsChannel <- SubsMessage{uri: req.Params.URI, action: unsubscribe , ss: ss}
			return nil
		},
		RootsListChangedHandler: func(ctx context.Context, req *mcp.RootsListChangedRequest) {
			log("- Client roots list changed")
			// We can't get the ServerSession from the request anymore, so we'll need to handle this differently
			_, _ = req.Session.ListRoots(ctx, &mcp.ListRootsParams{})
		},
		CompletionHandler: nil,
		InitializedHandler: func(_ context.Context, _ *mcp.InitializedRequest) {
			log("- Client initialized")
		},
		HasPrompts:   true,
		HasResources: true,
		HasTools:     true,
	})

	// Add interceptor middleware to the server (includes telemetry)
	middlewares := interceptors.Callbacks(
		g.LogCalls,
		g.BlockSecrets,
		g.OAuthInterceptorEnabled,
		parsedInterceptors,
	)
	if len(middlewares) > 0 {
		g.mcpServer.AddReceivingMiddleware(middlewares...)
	}

	// Which docker images are used?
	// Pull them and verify them if possible.
	if !g.Static {
		if err := g.pullAndVerify(ctx, configuration); err != nil {
			return err
		}

		// When running in a container, find on which network we are running.
		if os.Getenv("DOCKER_MCP_IN_CONTAINER") == "1" {
			networks, err := g.guessNetworks(ctx)
			if err != nil {
				return fmt.Errorf("guessing network: %w", err)
			}
			g.clientPool.SetNetworks(networks)
		}
	}

	if err := g.reloadConfiguration(ctx, configuration, nil, nil); err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Central mode.
	if g.Central {
		log("> Initialized (in central mode) in", time.Since(start))
		if g.DryRun {
			log("Dry run mode enabled, not starting the server.")
			return nil
		}

		log("> Start streaming server on port", g.Port)
		return g.startCentralStreamingServer(ctx, ln, configuration)
	}

	// Optionally watch for configuration updates.
	if configurationUpdates != nil {
		log("- Watching for configuration updates...")
		go func() {
			for {
				select {
				case <-ctx.Done():
					log("> Stop watching for updates")
					return
				case configuration := <-configurationUpdates:
					log("> Configuration updated, reloading...")

					if err := g.pullAndVerify(ctx, configuration); err != nil {
						logf("> Unable to pull and verify images: %s", err)
						continue
					}

					if err := g.reloadConfiguration(ctx, configuration, nil, nil); err != nil {
						logf("> Unable to list capabilities: %s", err)
						continue
					}
				}
			}
		}()
	}

	log("> Initialized in", time.Since(start))
	if g.DryRun {
		log("Dry run mode enabled, not starting the server.")
		return nil
	}

	// Start the server
	switch strings.ToLower(g.Transport) {
	case "stdio":
		log("> Start stdio server")
		return g.startStdioServer(ctx, os.Stdin, os.Stdout)

	case "sse":
		log("> Start sse server on port", g.Port)
		return g.startSseServer(ctx, ln)

	case "http", "streamable", "streaming", "streamable-http":
		log("> Start streaming server on port", g.Port)
		return g.startStreamingServer(ctx, ln)

	default:
		return fmt.Errorf(
			"unknown transport %q, expected 'stdio', 'sse' or 'streaming",
			g.Transport,
		)
	}
}

func (g *Gateway) reloadConfiguration(
	ctx context.Context,
	configuration Configuration,
	serverNames []string,
	clientConfig *clientConfig,
) error {
	defer func() {
		if r := recover(); r != nil {
			// Use fmt.Printf for logging since log is redefined as a function variable below
			fmt.Printf("Panic during configuration reload: %v\n", r)
		}
	}()

	// Handle logging function - use fmt.Printf as default
	log := func(msgs ...any) {
		fmt.Println(msgs...)
	}

	// If serverNames is nil, get them from configuration
	if serverNames == nil {
		serverNames = configuration.ServerNames()
	}

	if len(serverNames) == 0 {
		log("- No server is enabled")
	} else {
		log("- Those servers are enabled:", strings.Join(serverNames, ", "))
	}

	// Get capabilities from all enabled servers
	capabilities, err := g.listCapabilities(ctx, configuration, serverNames, clientConfig)
	if err != nil {
		return fmt.Errorf("failed to list capabilities: %w", err)
	}

	// Cleanup existing capabilities before adding new ones
	// This prevents accumulation of capabilities on reload
	if g.registeredToolNames != nil {
		g.mcpServer.RemoveTools(g.registeredToolNames...)
	}
	if g.registeredPromptNames != nil {
		g.mcpServer.RemovePrompts(g.registeredPromptNames...)
	}
	if g.registeredResourceURIs != nil {
		g.mcpServer.RemoveResources(g.registeredResourceURIs...)
	}
	if g.registeredResourceTemplateURIs != nil {
		g.mcpServer.RemoveResourceTemplates(g.registeredResourceTemplateURIs...)
	}

	// Reset tracking slices
	g.registeredToolNames = nil
	g.registeredPromptNames = nil
	g.registeredResourceURIs = nil
	g.registeredResourceTemplateURIs = nil

	// Add new capabilities and track them
	for _, tool := range capabilities.Tools {
		g.mcpServer.AddTool(tool.Tool, tool.Handler)
		g.registeredToolNames = append(g.registeredToolNames, tool.Tool.Name)
	}

	// Prompts are handled directly with AddPrompt in SDK v0.5.0
	for _, prompt := range capabilities.Prompts {
		g.mcpServer.AddPrompt(prompt.Prompt, prompt.Handler)
		g.registeredPromptNames = append(g.registeredPromptNames, prompt.Prompt.Name)
	}

	// Resources are handled directly
	for _, resource := range capabilities.Resources {
		g.mcpServer.AddResource(resource.Resource, resource.Handler)
		g.registeredResourceURIs = append(g.registeredResourceURIs, resource.Resource.URI)
	}

	// Resource templates - use the original URI template directly
	// SDK v0.5.0 includes proper RFC 6570 URI template support
	for _, template := range capabilities.ResourceTemplates {
		// Use the original URI template directly - SDK v0.5.0 handles RFC 6570 templates
		resource := &mcp.ResourceTemplate{
			URITemplate: template.ResourceTemplate.URITemplate,
			Name:        template.ResourceTemplate.Name,
			Description: template.ResourceTemplate.Description,
			MIMEType:    template.ResourceTemplate.MIMEType,
		}
		g.mcpServer.AddResourceTemplate(resource, template.Handler)
		g.registeredResourceTemplateURIs = append(
			g.registeredResourceTemplateURIs,
			resource.URITemplate,
		)
	}

	g.health.SetHealthy()
	log("Successfully reloaded configuration with", len(capabilities.Tools), "tools,",
		len(capabilities.Prompts), "prompts,", len(capabilities.Resources), "resources,",
		len(capabilities.ResourceTemplates), "resource templates")
	return nil
}

// RefreshCapabilities implements the CapabilityRefresher interface
// This method updates the server's capabilities by reloading the configuration
func (g *Gateway) RefreshCapabilities(
	ctx context.Context,
	server *mcp.Server,
	serverSession *mcp.ServerSession,
) error {
	// Get current configuration
	configuration, _, _, err := g.configurator.Read(ctx)
	// hold on to current serverNames
	configuration.serverNames = g.configuration.serverNames
	// reset on Gateway
	g.configuration = configuration
	if err != nil {
		return fmt.Errorf("failed to read configuration: %w", err)
	}

	// Create a clientConfig to reuse the existing session for the server that triggered the notification
	clientConfig := &clientConfig{
		serverSession: serverSession,
		server:        server,
	}

	// Refresh all servers, but the clientPool will reuse the existing session for the one that matches
	serverNames := configuration.ServerNames()
	log(
		"- RefreshCapabilities called for session, refreshing servers:",
		strings.Join(serverNames, ", "),
	)

	err = g.reloadConfiguration(ctx, configuration, serverNames, clientConfig)
	if err != nil {
		log("! Failed to refresh capabilities:", err)
	} else {
		log("- RefreshCapabilities completed successfully")
	}
	return err
}

// GetSessionCache returns the cached information for a server session
func (g *Gateway) GetSessionCache(ss *mcp.ServerSession) *ServerSessionCache {
	g.sessionCacheMu.RLock()
	defer g.sessionCacheMu.RUnlock()
	return g.sessionCache[ss]
}

// RemoveSessionCache removes the cached information for a server session
func (g *Gateway) RemoveSessionCache(ss *mcp.ServerSession) {
	g.sessionCacheMu.Lock()
	defer g.sessionCacheMu.Unlock()
	delete(g.sessionCache, ss)
}

// ListRoots checks if client supports Roots, gets them, and caches the result
func (g *Gateway) ListRoots(ctx context.Context, ss *mcp.ServerSession) {
	// Check if client supports Roots and get them if available
	rootsResult, err := ss.ListRoots(ctx, nil)

	g.sessionCacheMu.Lock()
	defer g.sessionCacheMu.Unlock()

	// Get existing cache or create new one
	cache, exists := g.sessionCache[ss]
	if !exists {
		cache = &ServerSessionCache{}
		g.sessionCache[ss] = cache
	}

	if err != nil {
		log("- Client does not support roots or error listing roots:", err)
		cache.Roots = nil
	} else {
		log("- Client supports roots, found", len(rootsResult.Roots), "roots")
		for _, root := range rootsResult.Roots {
			log("  - Root:", root.URI)
		}
		cache.Roots = rootsResult.Roots
	}
	g.clientPool.UpdateRoots(ss, cache.Roots)
}

// periodicMetricExport periodically exports metrics for long-running gateways
// This addresses the critical issue where Docker CLI's ManualReader only exports on shutdown
func (g *Gateway) periodicMetricExport(ctx context.Context) {
	// Get interval from environment or use default
	intervalStr := os.Getenv("DOCKER_MCP_METRICS_INTERVAL")
	interval := 30 * time.Second
	if intervalStr != "" {
		if parsed, err := time.ParseDuration(intervalStr); err == nil {
			interval = parsed
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Get the meter provider to force flush metrics
	meterProvider := otel.GetMeterProvider()

	if os.Getenv("DOCKER_MCP_TELEMETRY_DEBUG") != "" {
		fmt.Fprintf(
			os.Stderr,
			"[MCP-TELEMETRY] Starting periodic metric export every %v\n",
			interval,
		)
	}

	for {
		select {
		case <-ctx.Done():
			if os.Getenv("DOCKER_MCP_TELEMETRY_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "[MCP-TELEMETRY] Stopping periodic metric export\n")
			}
			return
		case <-ticker.C:
			// Force metric export
			if mp, ok := meterProvider.(interface{ ForceFlush(context.Context) error }); ok {
				flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				if err := mp.ForceFlush(flushCtx); err != nil {
					if os.Getenv("DOCKER_MCP_TELEMETRY_DEBUG") != "" {
						fmt.Fprintf(os.Stderr, "[MCP-TELEMETRY] Periodic flush error: %v\n", err)
					}
				} else {
					if os.Getenv("DOCKER_MCP_TELEMETRY_DEBUG") != "" {
						fmt.Fprintf(os.Stderr, "[MCP-TELEMETRY] Periodic metric flush successful\n")
					}
				}
				cancel()
			} else if os.Getenv("DOCKER_MCP_TELEMETRY_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "[MCP-TELEMETRY] WARNING: MeterProvider does not support ForceFlush\n")
			}
		}
	}
}
