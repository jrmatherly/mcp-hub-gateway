package catalog

import (
	"context"
	"os"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/config"
)

func Reset(context.Context) error {
	catalogsDir, err := config.FilePath("catalogs")
	if err != nil {
		return err
	}
	if err := os.RemoveAll(catalogsDir); err != nil {
		return err
	}

	return WriteConfig(&Config{})
}
