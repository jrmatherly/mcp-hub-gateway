package policy

import (
	"context"
	"fmt"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/desktop"
)

func Dump(ctx context.Context) error {
	l, err := desktop.NewSecretsClient().GetJfsPolicy(ctx)
	if err != nil {
		return err
	}

	fmt.Println(l)
	return nil
}
