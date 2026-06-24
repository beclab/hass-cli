package cmd

import (
	"fmt"
	"os"

	"github.com/beclab/hass-cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newLovelaceCmd manages dashboards, their stored configs, and custom
// resources. Everything here is WebSocket-only. The default dashboard is
// addressed by a null url_path; named dashboards use their url_path.
func newLovelaceCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lovelace",
		Aliases: []string{"dashboard"},
		Short:   "Manage Lovelace dashboards, configs, and resources",
		Example: `  hass-cli lovelace dashboard list
  hass-cli lovelace config get --dashboard ops-room
  hass-cli lovelace config save --dashboard ops-room --file dash.yaml`,
	}
	cmd.AddCommand(newLovelaceDashboardCmd(f))
	cmd.AddCommand(newLovelaceConfigCmd(f))
	cmd.AddCommand(newLovelaceResourceCmd(f))
	return cmd
}

func newLovelaceDashboardCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "List and manage storage-mode dashboards",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List dashboards",
		Args:  cobra.NoArgs,
		RunE: wsRunCols(f, colsDashboard, func([]string) map[string]any {
			return map[string]any{"type": "lovelace/dashboards/list"}
		}),
	})

	var createData string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: `Create a dashboard (--data '{"url_path":"ops","title":"Ops"}')`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := parseDataObject(createData)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf(`--data is required (e.g. {"url_path":"ops","title":"Ops"})`)
			}
			body["type"] = "lovelace/dashboards/create"
			if _, ok := body["mode"]; !ok {
				body["mode"] = "storage"
			}
			return wsCall(f, cmd, body)
		},
	}
	createCmd.Flags().StringVar(&createData, "data", "", "Dashboard fields as JSON (or @file.json)")
	cmd.AddCommand(createCmd)

	var updateData string
	updateCmd := &cobra.Command{
		Use:   "update <dashboard_id>",
		Short: "Update a dashboard's mutable fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := requireData(updateData)
			if err != nil {
				return err
			}
			body["type"] = "lovelace/dashboards/update"
			body["dashboard_id"] = args[0]
			return wsCall(f, cmd, body)
		},
	}
	updateCmd.Flags().StringVar(&updateData, "data", "", "Fields to change as JSON (or @file.json)")
	cmd.AddCommand(updateCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <dashboard_id>",
		Short: "Delete a dashboard",
		Args:  cobra.ExactArgs(1),
		RunE: wsRun(f, func(args []string) map[string]any {
			return map[string]any{"type": "lovelace/dashboards/delete", "dashboard_id": args[0]}
		}),
	})

	return cmd
}

func newLovelaceConfigCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read, save, or delete a dashboard's stored config",
	}

	var getDash string
	var getForce bool
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a dashboard config (default dashboard unless --dashboard)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return wsCall(f, cmd, map[string]any{
				"type": "lovelace/config", "url_path": urlPathOrNil(getDash), "force": getForce,
			})
		},
	}
	getCmd.Flags().StringVar(&getDash, "dashboard", "", "Dashboard url_path (omit for default)")
	getCmd.Flags().BoolVar(&getForce, "force", false, "Force re-read from storage")
	cmd.AddCommand(getCmd)

	var saveDash, saveFile string
	saveCmd := &cobra.Command{
		Use:   "save",
		Short: "Save a dashboard config from a YAML/JSON file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if saveFile == "" {
				return fmt.Errorf("--file is required")
			}
			data, err := os.ReadFile(saveFile)
			if err != nil {
				return err
			}
			var config any
			if err := yaml.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("parse %s: %w", saveFile, err)
			}
			return wsCall(f, cmd, map[string]any{
				"type": "lovelace/config/save", "url_path": urlPathOrNil(saveDash), "config": config,
			})
		},
	}
	saveCmd.Flags().StringVar(&saveDash, "dashboard", "", "Dashboard url_path (omit for default)")
	saveCmd.Flags().StringVar(&saveFile, "file", "", "Config file (YAML or JSON)")
	cmd.AddCommand(saveCmd)

	var delDash string
	delCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a dashboard's stored config (reverts to auto-generated)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return wsCall(f, cmd, map[string]any{
				"type": "lovelace/config/delete", "url_path": urlPathOrNil(delDash),
			})
		},
	}
	delCmd.Flags().StringVar(&delDash, "dashboard", "", "Dashboard url_path (omit for default)")
	cmd.AddCommand(delCmd)

	return cmd
}

func newLovelaceResourceCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource",
		Short: "Manage custom dashboard resources (JS/CSS modules)",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List resources",
		Args:  cobra.NoArgs,
		RunE: wsRun(f, func([]string) map[string]any {
			return map[string]any{"type": "lovelace/resources"}
		}),
	})

	var createData string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: `Create a resource (--data '{"res_type":"module","url":"/local/x.js"}')`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := parseDataObject(createData)
			if err != nil {
				return err
			}
			if body == nil {
				return fmt.Errorf(`--data is required (e.g. {"res_type":"module","url":"/local/x.js"})`)
			}
			body["type"] = "lovelace/resources/create"
			return wsCall(f, cmd, body)
		},
	}
	createCmd.Flags().StringVar(&createData, "data", "", "Resource fields as JSON (or @file.json)")
	cmd.AddCommand(createCmd)

	var updateData string
	updateCmd := &cobra.Command{
		Use:   "update <resource_id>",
		Short: "Update a resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := requireData(updateData)
			if err != nil {
				return err
			}
			body["type"] = "lovelace/resources/update"
			body["resource_id"] = args[0]
			return wsCall(f, cmd, body)
		},
	}
	updateCmd.Flags().StringVar(&updateData, "data", "", "Fields to change as JSON (or @file.json)")
	cmd.AddCommand(updateCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <resource_id>",
		Short: "Delete a resource",
		Args:  cobra.ExactArgs(1),
		RunE: wsRun(f, func(args []string) map[string]any {
			return map[string]any{"type": "lovelace/resources/delete", "resource_id": args[0]}
		}),
	})

	return cmd
}

// urlPathOrNil maps an empty dashboard flag to a null url_path, which targets
// the default dashboard in the Lovelace WS API.
func urlPathOrNil(p string) any {
	if p == "" {
		return nil
	}
	return p
}
