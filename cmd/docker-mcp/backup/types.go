package backup

import "github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/desktop"

type Backup struct {
	Config       string            `json:"config"`
	Registry     string            `json:"registry"`
	Catalog      string            `json:"catalog"`
	CatalogFiles map[string]string `json:"catalogFiles"`
	Tools        string            `json:"tools"`
	Secrets      []desktop.Secret  `json:"secrets"`
	Policy       string            `json:"policy"`
}
