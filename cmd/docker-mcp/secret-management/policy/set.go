package policy

import (
	"context"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/desktop"
)

func Set(ctx context.Context, data string) error {
	return desktop.NewSecretsClient().SetJfsPolicy(ctx, data)
}
