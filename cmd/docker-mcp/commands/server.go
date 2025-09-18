package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/config"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/docker"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/oci"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/server"
)

func serverCommand(docker docker.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage servers",
	}

	var outputJSON bool
	lsCommand := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List enabled servers",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			list, err := server.List(cmd.Context(), docker)
			if err != nil {
				return err
			}

			if outputJSON {
				buf, err := json.Marshal(list)
				if err != nil {
					return err
				}
				_, _ = cmd.OutOrStdout().Write(buf)
			} else if len(list) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No server is enabled")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), strings.Join(list, ", "))
			}

			return nil
		},
	}
	lsCommand.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")
	cmd.AddCommand(lsCommand)

	cmd.AddCommand(&cobra.Command{
		Use:     "enable",
		Aliases: []string{"add"},
		Short:   "Enable a server or multiple servers",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.Enable(cmd.Context(), docker, args)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "disable",
		Aliases: []string{"remove", "rm"},
		Short:   "Disable a server or multiple servers",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.Disable(cmd.Context(), docker, args)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "inspect",
		Short: "Get information about a server or inspect an OCI artifact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := args[0]

			// Check if the argument looks like an OCI reference
			// OCI refs typically contain a registry/repository pattern with optional tag or digest
			if strings.Contains(arg, "/") &&
				(strings.Contains(arg, ":") || strings.Contains(arg, "@")) {
				// Use OCI inspect for OCI references
				return oci.InspectArtifact(arg)
			}

			// Use regular server inspect for server names
			info, err := server.Inspect(cmd.Context(), docker, arg)
			if err != nil {
				return err
			}

			buf, err := info.ToJSON()
			if err != nil {
				return err
			}

			_, _ = cmd.OutOrStdout().Write(buf)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Disable all the servers",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return config.WriteRegistry(nil)
		},
	})

	return cmd
}
