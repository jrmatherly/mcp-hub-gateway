package oauth

import (
	"context"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/desktop"
)

func Revoke(ctx context.Context, app string) error {
	client := desktop.NewAuthClient()

	return client.DeleteOAuthApp(ctx, app)
}
